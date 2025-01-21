package main

import (
	"fmt"
	"sync"
)

type Task func()

type ThreadPool struct {
	corePoolSize int			// 核心线程数
	maxPoolSize  int			// 最大线程数
	taskQueue    chan Task		// 任务队列
	workerQueue	 chan struct{}	// 线程队列
	mu           sync.Mutex		// 锁，用于并发控制
	wg			 sync.WaitGroup // 类似信号量，避免spin带来的资源损耗
	shutdown     chan struct{}	// 关闭信号
}

// NewThreadPool 创建线程池
func NewThreadPool(corePoolSize int, maxPoolSize int, queueSize int) *ThreadPool {
	// 1. 输入参数校验
	if corePoolSize < 1 || maxPoolSize < 1 || queueSize < 0 {
		panic("输入线程数和队列大小不合法")
	}
	if corePoolSize > maxPoolSize {
		panic("核心线程数应小于等于最大线程数")
	}
	// 2. 创建线程池
	tp := &ThreadPool{
		corePoolSize: corePoolSize,
		maxPoolSize:  maxPoolSize,
		taskQueue:    make(chan Task, queueSize),
		workerQueue:  make(chan struct{}, maxPoolSize),
		shutdown:     make(chan struct{}),
	}
	// 3. 创建核心线程
	for i := 0; i < corePoolSize; i ++ {
		// 3.1. 记录创建的线程，确保在关闭时不遗漏
		tp.wg.Add(1)
		// 3.2. 创建线程并启动
		go tp.worker()
	}

	return tp
}

func (tp *ThreadPool) worker() {
	// 1. 核心线程不停运行，并对任务队列和关闭信号进行监听
	defer tp.wg.Done()
	for {
		select {
		// 2. 若任务队列有任务，则取出任务并执行
		case task, ok := <-tp.taskQueue:
			if !ok {
				return
			}
			task()
		// 3. 若关闭信号发出，则退出线程
		case <-tp.shutdown:
			return
		}
	}
}

// submit 提交任务
func (tp *ThreadPool) submit(task Task) {
	select {
	// 1. 若线程池未关闭，则将任务放入任务队列
	case tp.taskQueue <- task:
	// 2. 若线程池已关闭，则丢弃任务
	case <-tp.shutdown:
		return
	// 3. 若任务队列已满，则创建新的线程
	default:
		select {
		case tp.workerQueue <- struct{}{}:
			tp.wg.Add(1)
			go tp.worker()
			tp.taskQueue <- task
		// 4. 无法创建新线程，任务被拒绝
		default:
			fmt.Println("Task rejected: queue is full and no more threads can be created")
		}
	}
}

// Shutdown 关闭线程池
func (p *ThreadPool) Shutdown() {
	close(p.shutdown)
	p.wg.Wait()
	close(p.taskQueue)
}
