#include <atomic>
#include <condition_variable>
#include <functional>
#include <iostream>
#include <mutex>
#include <queue>
#include <thread>
#include <vector>

class ThreadPool {
 public:
    ThreadPool(size_t corePoolSize, size_t maxPoolSize, size_t queueSize)
        // 1. 初始化线程池基本参数
        : corePoolSize_(corePoolSize), maxPoolSize_(maxPoolSize), queueSize_(queueSize), isShutdown_(false) {
            // 2. 创建核心线程
            for (size_t i = 0; i < corePoolSize_; i ++) {
                workers_.emplace_back([this]() -> void { this->workerThread(); });
            }
        }

    ~ThreadPool() {
        Shutdown();
    }

    void Submit(std::function<void()> task) {
        std::unique_lock<std::mutex> lock(mtx_);
        if (isShutdown_) {
            std::cerr << "线程池已关闭，不能提交任务" << std::endl;
        }
        if (tasks_.size() >= queueSize_ && workers_.size() >= maxPoolSize_) {
            std::cerr << "已达到最大线程数，不能提交任务" << std::endl;
        }
        if (tasks_.size() >= queueSize_ && workers_.size() < maxPoolSize_) {
            workers_.emplace_back([this] { this->workerThread(); });
        }
        tasks_.push(task);
        lock.unlock();
        cv_.notify_one();
    }

    void Shutdown() {
        if (isShutdown_) {
            return;
        }
        isShutdown_ = true;
        cv_.notify_all();
        for (auto& worker : workers_) {
            if (worker.joinable()) {
                worker.join();
            }
        }
    }

 private:
    void workerThread() {
        while (!isShutdown_) {
            std::function<void()> task;
            std::unique_lock<std::mutex> lock(mtx_);
            cv_.wait(lock, [this]() -> bool { return !tasks_.empty() || isShutdown_; });
            if (isShutdown_ || tasks_.empty()) {
                return;
            }
            task = tasks_.front();
            tasks_.pop();
            lock.unlock();
            task();
        }
    }

    size_t corePoolSize_;                       // 核心线程数
    size_t maxPoolSize_;                        // 最大线程数
    size_t queueSize_;                          // 任务队列大小 
    std::vector<std::thread> workers_;          // 工作线程集合
    std::queue<std::function<void()>> tasks_;   // 任务队列
    std::mutex mtx_;                            // 互斥锁
    std::condition_variable cv_;                // 条件变量
    std::atomic<bool> isShutdown_;              // 线程池是否关闭
};
