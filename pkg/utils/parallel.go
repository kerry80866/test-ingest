package utils

import (
	"sync"
)

type TaskResult struct {
	ID     uint64
	Result any   // 真正的业务结果
	Error  error // 处理过程中是否出错
}

// ParallelMap 并行处理输入切片中的元素，保持结果顺序与输入一致
// T: 输入元素类型
// R: 输出结果类型
// inputs: 待处理的输入元素切片
// workerNum: 并发工作协程数量，实际并发数不会超过输入元素数量
// fn: 处理单个元素的函数
// 返回: 与输入顺序一致的处理结果切片
func ParallelMap[T any, R any, C any](
	inputs []T,
	workerNum int,
	contextFactory func() C,
	fn func(ctx C, input T) R,
) []R {
	inputLen := len(inputs)
	// 空输入快速返回
	if inputLen == 0 {
		return nil
	}

	// 单个元素时直接处理，避免并发开销
	if inputLen == 1 {
		return []R{fn(contextFactory(), inputs[0])}
	}

	// 限制工作协程数量不超过输入元素数
	workerNum = min(inputLen, workerNum)

	var wg sync.WaitGroup
	// 任务输入通道: 包含索引和值，用于任务分发
	inputCh := make(chan struct {
		index int
		value T
	})
	// 避免大输入时占用过多内存，同时减少阻塞
	resultCh := make(chan struct {
		index  int
		result R
	}, inputLen)

	// 启动工作协程
	for i := 0; i < workerNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := contextFactory()

			// 循环处理输入通道中的任务
			for in := range inputCh {
				result := fn(ctx, in.value)
				resultCh <- struct {
					index  int
					result R
				}{in.index, result}
			}
		}()
	}

	// 将输入数据发送到任务通道
	go func() {
		for i, v := range inputs {
			inputCh <- struct {
				index int
				value T
			}{i, v}
		}
		// 关闭输入通道，通知工作协程没有更多任务
		close(inputCh)
	}()

	// 等待所有工作协程完成，然后关闭结果通道
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 按原始顺序收集结果
	results := make([]R, inputLen)
	for r := range resultCh {
		results[r.index] = r.result
	}

	return results
}
