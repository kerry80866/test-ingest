package ingest

import (
	"context"
	"database/sql"
	"dex-ingest-sol/internal/config"
	"dex-ingest-sol/internal/ingest/handler"
	"dex-ingest-sol/internal/model"
	"dex-ingest-sol/internal/pkg/logger"
	"dex-ingest-sol/internal/pkg/utils"
	"dex-ingest-sol/pb"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/hashicorp/golang-lru"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
	"runtime/debug"
	"sync"
	"time"
)

// BLOCKBATCH_BUFFER 表示每个分区缓存的最大 batch 数
const BLOCKBATCH_BUFFER = 10

// BlockBatch 表示按 slot 聚合的一批数据，包含事件和对象
type BlockBatch struct {
	Slot     uint64
	IsGrpc   bool                // 是否为 gRPC 实时推送（false 表示补块）
	Messages []*kafka.Message    // Kafka 原始消息
	Balances []*model.Balance    // 构建的链上事件
	Events   []*model.ChainEvent // 构建的链上事件
	Pools    []*model.Pool       // 新增池子（如建池）
	Tokens   []*model.Token      // 新增 token（如首次初始化）
}

// WorkerContext 表示每个分区独立处理上下文
type WorkerContext struct {
	RouterType  RouterType            // 消费者类型（事件 / 余额）
	Partition   int32                 // 当前分区编号
	DB          *sql.DB               // 数据库连接
	Redis       *redis.Client         // Redis 客户端
	MsgCh       <-chan *kafka.Message // Kafka 消息通道
	Kafka       *kafka.Consumer       // Kafka 消费者（用于 commit）
	Base58Cache *lru.Cache            // base58 解码缓存
	BatchQueue  []*BlockBatch         // 当前缓存的 batch 队列

	// 配置项
	MaxBlockHold  int           // 达到 N 条 block 时强制 flush
	MaxBatchFlush int           // 每次最多 flush M 条 batch
	FlushInterval time.Duration // flush 超时时间

	// 内部状态
	ctx           context.Context
	flushCounter  int
	lastFlushTime time.Time
	lastSlot      uint64
}

// StartWorker 启动分区 worker，处理指定类型的消息
func StartWorker(
	ctx context.Context,
	partition int32,
	ch <-chan *kafka.Message,
	db *sql.DB,
	redis *redis.Client,
	kafkaConsumer *kafka.Consumer,
	conf *config.WorkerConfig,
	routerType RouterType,
) {
	w := &WorkerContext{
		ctx:           ctx,
		RouterType:    routerType,
		Partition:     partition,
		DB:            db,
		Redis:         redis,
		MsgCh:         ch,
		Kafka:         kafkaConsumer,
		Base58Cache:   utils.NewBase58Cache(),
		BatchQueue:    make([]*BlockBatch, 0, BLOCKBATCH_BUFFER),
		MaxBlockHold:  conf.MaxBlockHold,
		MaxBatchFlush: conf.MaxBatchFlush,
		FlushInterval: conf.FlushInterval,
		lastFlushTime: time.Now(),
		flushCounter:  0,
	}
	w.Run()
}

func (w *WorkerContext) Run() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			logger.Infof("[partition=%d] worker exiting", w.Partition)
			return

		case msg, ok := <-w.MsgCh:
			if !ok {
				logger.Infof("[partition=%d] channel closed", w.Partition)
				return
			}
			w.handleMessage(msg)

		case <-ticker.C:
			// 超过 flush 间隔才触发兜底
			if len(w.BatchQueue) > 0 && time.Since(w.lastFlushTime) > w.FlushInterval {
				w.flushIfNeeded()
			}
		}
	}
}

func (w *WorkerContext) handleMessage(msg *kafka.Message) {
	batch := buildBlockBatch(w.RouterType, w.Partition, msg, w.Base58Cache)
	if batch != nil {
		w.BatchQueue = append(w.BatchQueue, batch)
	}

	if len(w.BatchQueue) >= w.MaxBlockHold {
		w.flushIfNeeded()
	}
}

func (w *WorkerContext) flushIfNeeded() {
	if len(w.BatchQueue) == 0 {
		return
	}

	flushCount := min(len(w.BatchQueue), w.MaxBatchFlush)
	toFlush := w.BatchQueue[:flushCount]

	// 实际批处理逻辑
	maxSlot := w.flushBatches(toFlush)

	// 更新状态
	w.flushCounter++
	w.lastFlushTime = time.Now()
	if maxSlot > w.lastSlot {
		logger.Infof("[partition=%d] slot advanced: %d → %d", w.Partition, w.lastSlot, maxSlot)
		w.lastSlot = maxSlot
	}

	// 内存控制：每 100 次强制复制剩余数据，防止底层数组持续增长
	if w.flushCounter%100 == 0 {
		remain := len(w.BatchQueue) - flushCount
		if remain < 0 {
			remain = 0
		}
		newCap := remain + BLOCKBATCH_BUFFER
		w.BatchQueue = append(make([]*BlockBatch, 0, newCap), w.BatchQueue[flushCount:]...)
	} else {
		w.BatchQueue = w.BatchQueue[flushCount:]
	}

	// 打印 flush 范围日志（可观察 slot 分布）
	if flushCount > 0 {
		startSlot := toFlush[0].Slot
		endSlot := toFlush[len(toFlush)-1].Slot

		// 统计事件数量（事件 Router 才有意义）
		eventCount := 0
		if w.RouterType == RouterEvent {
			for _, b := range toFlush {
				eventCount += len(b.Events)
			}
		}

		logger.Infof("[partition=%d] flushing %d batches, %d events (slots %d → %d)",
			w.Partition, flushCount, eventCount, startSlot, endSlot)
	}
}

func buildBlockBatch(routerType RouterType, partition int32, msg *kafka.Message, cache *lru.Cache) *BlockBatch {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf(
				"[router=%v partition=%d offset=%d topic=%s] buildBlockBatch panic: %v\n%s",
				routerType,
				partition,
				msg.TopicPartition.Offset,
				*msg.TopicPartition.Topic,
				r,
				debug.Stack(),
			)
		}
	}()

	events := &pb.Events{}
	err := proto.Unmarshal(msg.Value, events)
	if err != nil {
		logger.Errorf("failed to unmarshal kafka message: %v", err)
		return nil
	}
	if len(events.Events) == 0 {
		return nil
	}

	batch := &BlockBatch{
		Slot:     events.Slot,
		IsGrpc:   events.Source == 1,
		Messages: []*kafka.Message{msg},
	}

	// 根据 RouterType 分发事件解析
	switch routerType {
	case RouterEvent:
		batch.Events = handler.BuildChainEventModels(events, cache)
		batch.Pools = handler.BuildPoolModels(events, cache)
		batch.Tokens = handler.BuildTokenModels(events, cache)
	case RouterBalance:
		batch.Balances = handler.BuildBalanceModels(events, cache)
	}

	return batch
}

func (w *WorkerContext) flushBatches(batches []*BlockBatch) uint64 {
	switch w.RouterType {
	case RouterEvent:
		w.flushEventBatches(batches)
	case RouterBalance:
		w.flushBalanceBatches(batches)
	default:
		logger.Errorf("[partition=%d] unknown RouterType: %v", w.Partition, w.RouterType)
	}

	var maxSlot uint64
	for _, b := range batches {
		if b.Slot > maxSlot {
			maxSlot = b.Slot
		}
	}
	return maxSlot
}

func (w *WorkerContext) flushEventBatches(batches []*BlockBatch) {
	var eventCount, poolCount, tokenCount int

	// 第一次遍历：统计容量
	for _, b := range batches {
		eventCount += len(b.Events)
		poolCount += len(b.Pools)
		tokenCount += len(b.Tokens)
	}

	// 分配内存
	allEvents := make([]*model.ChainEvent, 0, eventCount)
	pools := make([]*model.Pool, 0, poolCount)
	tokens := make([]*model.Token, 0, tokenCount)

	// 第二次遍历：聚合数据
	for _, b := range batches {
		allEvents = append(allEvents, b.Events...)
		pools = append(pools, b.Pools...)
		tokens = append(tokens, b.Tokens...)
	}

	var wg sync.WaitGroup

	// 写入 ChainEvent
	if len(allEvents) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			if err := handler.InsertChainEvents(w.ctx, w.DB, allEvents); err != nil {
				logger.Errorf("[partition=%d] insertEvents error: %v", w.Partition, err)
			}
			logger.Infof("[partition=%d] insertChainEvents done in %s", w.Partition, time.Since(start))
		}()

		// Redis 同步缓存（如启用）
		if w.Redis != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := handler.SyncPoolCache(w.ctx, w.Redis, allEvents); err != nil {
					logger.Errorf("[partition=%d] sync pool cache error: %v", w.Partition, err)
				}
			}()
		}
	}

	// 写入 Pool 数据
	if len(pools) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			if err := handler.InsertPools(w.ctx, w.DB, pools); err != nil {
				logger.Errorf("[partition=%d] insertPools error: %v", w.Partition, err)
			}
			logger.Infof("[partition=%d] insertPools done in %s", w.Partition, time.Since(start))
		}()
	}

	// 写入 Token 数据
	if len(tokens) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			if err := handler.InsertTokens(w.ctx, w.DB, tokens); err != nil {
				logger.Errorf("[partition=%d] insertTokens error: %v", w.Partition, err)
			}
			logger.Infof("[partition=%d] insertTokens done in %s", w.Partition, time.Since(start))
		}()
	}

	// 等待所有任务完成
	wg.Wait()

	// 提交 Kafka offset
	w.commitLastMessage(batches)
}

func (w *WorkerContext) flushBalanceBatches(batches []*BlockBatch) {
	var (
		totalBalanceCount              int
		realtimeCount, historicalCount int
	)

	for _, b := range batches {
		n := len(b.Balances)
		totalBalanceCount += n
		if b.isRealtime(w.lastSlot) {
			realtimeCount += n
		} else {
			historicalCount += n
		}
	}
	if totalBalanceCount == 0 {
		return
	}

	realtimeBalances := make([]*model.Balance, 0, realtimeCount)
	historicalBalances := make([]*model.Balance, 0, historicalCount)

	for _, b := range batches {
		if b.isRealtime(w.lastSlot) {
			realtimeBalances = append(realtimeBalances, b.Balances...)
		} else {
			historicalBalances = append(historicalBalances, b.Balances...)
		}
	}

	logger.Infof("[partition=%d] flushing %d balances (realtime: %d, historical: %d)",
		w.Partition, totalBalanceCount, len(realtimeBalances), len(historicalBalances))

	if len(realtimeBalances) > 0 {
		start := time.Now()
		if err := handler.InsertBalances(w.ctx, w.DB, realtimeBalances, true); err != nil {
			logger.Errorf("[partition=%d] insertBalances (realtime) error: %v", w.Partition, err)
		}
		logger.Infof("[partition=%d] insertBalances (realtime) done in %s", w.Partition, time.Since(start))
	}

	if len(historicalBalances) > 0 {
		start := time.Now()
		if err := handler.InsertBalances(w.ctx, w.DB, historicalBalances, false); err != nil {
			logger.Errorf("[partition=%d] insertBalances (historical) error: %v", w.Partition, err)
		}
		logger.Infof("[partition=%d] insertBalances (historical) done in %s", w.Partition, time.Since(start))
	}

	w.commitLastMessage(batches)
}

func (w *WorkerContext) commitLastMessage(batches []*BlockBatch) {
	if len(batches) == 0 {
		return
	}
	lastBatch := batches[len(batches)-1]
	if len(lastBatch.Messages) > 0 {
		lastMsg := lastBatch.Messages[len(lastBatch.Messages)-1]
		if _, err := w.Kafka.CommitMessage(lastMsg); err != nil {
			logger.Errorf("[partition=%d] commit message failed: %v", w.Partition, err)
		}
	}
}

func (b *BlockBatch) isRealtime(lastSlot uint64) bool {
	if lastSlot == 0 {
		return b.IsGrpc
	}
	return b.Slot > lastSlot
}
