package ingest

import (
	"context"
	"database/sql"
	"dex-ingest-sol/internal/config"
	"dex-ingest-sol/pkg/logger"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/redis/go-redis/v9"
	"sync"
)

type PartitionRouter struct {
	mu      sync.Mutex
	workers map[int32]chan *kafka.Message
	wg      sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc

	routerType RouterType
	db         *sql.DB
	redis      *redis.Client
	kafka      *kafka.Consumer
	config     *config.WorkerConfig
}

// NewPartitionRouter 构造函数，需指定类型
func NewPartitionRouter(db *sql.DB, redis *redis.Client, cfg *config.WorkerConfig, routerType RouterType) *PartitionRouter {
	ctx, cancel := context.WithCancel(context.Background())
	return &PartitionRouter{
		workers:    make(map[int32]chan *kafka.Message),
		ctx:        ctx,
		cancel:     cancel,
		routerType: routerType,
		db:         db,
		redis:      redis,
		config:     cfg,
	}
}

func (r *PartitionRouter) SetKafkaConsumer(k *kafka.Consumer) {
	r.kafka = k
}

// Start 空实现，兼容 go-zero Service 接口
func (r *PartitionRouter) Start() {
	logger.Infof("PartitionRouter started: type=%v", r.routerType)
}

// Stop 优雅关闭所有 worker
func (r *PartitionRouter) Stop() {
	r.cancel()
	r.mu.Lock()
	for _, ch := range r.workers {
		close(ch)
	}
	r.mu.Unlock()
	r.wg.Wait()
}

// Dispatch 根据分区分发消息
func (r *PartitionRouter) Dispatch(msg *kafka.Message) {
	partition := msg.TopicPartition.Partition

	r.mu.Lock()
	ch, ok := r.workers[partition]
	if !ok {
		ch = make(chan *kafka.Message, 1000)
		r.workers[partition] = ch
		r.wg.Add(1)

		go func(partition int32, ch <-chan *kafka.Message) {
			defer r.wg.Done()
			StartWorker(r.ctx, partition, ch, r.db, r.redis, r.kafka, r.config, r.routerType)
		}(partition, ch)
	}
	r.mu.Unlock()

	select {
	case ch <- msg:
	default:
		logger.Warnf("[partition=%d] message dropped or blocked", partition)
		// 或者 block 但打日志
		ch <- msg
	}
}
