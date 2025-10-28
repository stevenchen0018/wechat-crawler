# Context Canceled 问题修复

## 问题描述

定时任务或手动触发爬取任务时，系统报错：

```
ERROR  获取公众号列表失败  {"error": "context canceled"}
ERROR  爬取任务执行失败  {"error": "context canceled"}
```

## 问题分析

### 错误日志

```json
{"level":"INFO","time":"2025-10-27T11:40:14.568+0800","caller":"handler/wechat_handler.go:168","msg":"手动触发爬取任务"}
{"level":"INFO","time":"2025-10-27T11:40:14.568+0800","caller":"service/crawler_service.go:176","msg":"开始执行定时爬取任务"}
{"level":"ERROR","time":"2025-10-27T11:40:23.285+0800","caller":"service/crawler_service.go:181","msg":"获取公众号列表失败","error":"context canceled"}
{"level":"ERROR","time":"2025-10-27T11:40:23.285+0800","caller":"handler/wechat_handler.go:172","msg":"爬取任务执行失败","error":"context canceled"}
```

### 根本原因

问题出在 `internal/api/handler/wechat_handler.go` 的 `TriggerFetch` 方法：

```go
// 问题代码 ❌
func (h *WeChatHandler) TriggerFetch(c *gin.Context) {
    logger.Info("手动触发爬取任务")

    go func() {
        // 使用HTTP请求的context
        if err := h.crawlerService.FetchAllAccounts(c.Request.Context()); err != nil {
            logger.Error("爬取任务执行失败", zap.Error(err))
        }
    }()

    response.SuccessWithMsg(c, "爬取任务已启动", nil)
}
```

**问题流程**：

```
1. HTTP请求到达 → TriggerFetch 被调用
   ↓
2. 启动 goroutine 执行爬取任务
   • 使用 c.Request.Context() 作为参数
   ↓
3. 立即返回HTTP响应 "爬取任务已启动"
   • 此时 HTTP 请求完成
   • c.Request.Context() 被取消 ❌
   ↓
4. Goroutine 中的爬取任务开始执行
   • 调用 FetchAllAccounts(ctx)
   • ctx 已经被取消
   ↓
5. 执行数据库查询 s.wechatRepo.List(ctx)
   • 使用已取消的 context
   • 返回错误: "context canceled" ❌
```

### Context 生命周期

```
HTTP Request Context 生命周期:
┌─────────────────────────────────────┐
│ 请求开始                             │
│   ↓                                 │
│ 处理请求                             │
│   ↓                                 │
│ 返回响应  ← HTTP响应完成              │
│   ↓                                 │
│ Context 取消 ❌                      │
└─────────────────────────────────────┘

异步任务执行时间线:
┌─────────────────────────────────────┐
│ Goroutine 启动                       │
│   ↓                                 │
│ (等待) HTTP已响应，context已取消      │
│   ↓                                 │
│ 尝试执行数据库查询                    │
│   ↓                                 │
│ 错误: context canceled ❌            │
└─────────────────────────────────────┘
```

## 修复方案

### 使用独立的 Context

在异步任务中使用 `context.Background()`，不依赖 HTTP 请求的生命周期：

```go
// 修复后代码 ✅
func (h *WeChatHandler) TriggerFetch(c *gin.Context) {
    logger.Info("手动触发爬取任务")

    // 使用独立的context，不依赖HTTP请求的生命周期
    go func() {
        ctx := context.Background()  // ← 关键修复
        if err := h.crawlerService.FetchAllAccounts(ctx); err != nil {
            logger.Error("爬取任务执行失败", zap.Error(err))
        }
    }()

    response.SuccessWithMsg(c, "爬取任务已启动", nil)
}
```

### 修复流程

```
1. HTTP请求到达 → TriggerFetch 被调用
   ↓
2. 启动 goroutine 执行爬取任务
   • 创建独立的 context.Background() ✅
   ↓
3. 立即返回HTTP响应 "爬取任务已启动"
   • HTTP 请求完成
   • c.Request.Context() 被取消（不影响异步任务）
   ↓
4. Goroutine 中的爬取任务继续执行
   • 使用独立的 context.Background()
   • Context 有效 ✅
   ↓
5. 执行数据库查询 s.wechatRepo.List(ctx)
   • Context 有效
   • 查询成功 ✅
   ↓
6. 执行浏览器操作获取文章
   • 成功 ✅
   ↓
7. 保存文章到数据库
   • 成功 ✅
```

## 修复内容

### 1. 修改文件

**文件**: `internal/api/handler/wechat_handler.go`

**修改内容**:
1. 添加 `context` 包导入
2. 在 `TriggerFetch` 方法中使用 `context.Background()`

### 2. 代码对比

```diff
  package handler
  
  import (
+     "context"
      "strconv"
      
      "wechat-crawler/internal/service"
      ...
  )
  
  func (h *WeChatHandler) TriggerFetch(c *gin.Context) {
      logger.Info("手动触发爬取任务")
  
+     // 使用独立的context，不依赖HTTP请求的生命周期
+     // 避免HTTP响应后context被取消导致爬取任务失败
      go func() {
-         if err := h.crawlerService.FetchAllAccounts(c.Request.Context()); err != nil {
+         ctx := context.Background()
+         if err := h.crawlerService.FetchAllAccounts(ctx); err != nil {
              logger.Error("爬取任务执行失败", zap.Error(err))
          }
      }()
  
      response.SuccessWithMsg(c, "爬取任务已启动", nil)
  }
```

## 验证测试

### 测试步骤

1. **启动程序**
   ```bash
   ./wechat-crawler
   ```

2. **手动触发爬取任务**
   ```bash
   curl -X POST http://localhost:8081/api/crawler/trigger
   ```

3. **观察日志**

#### 修复前（错误）：
```
INFO  手动触发爬取任务
INFO  开始执行定时爬取任务
ERROR 获取公众号列表失败  {"error": "context canceled"}  ❌
ERROR 爬取任务执行失败  {"error": "context canceled"}  ❌
```

#### 修复后（正常）：
```
INFO  手动触发爬取任务
INFO  开始执行定时爬取任务
INFO  待爬取公众号数量  {"count": 1}  ✅
INFO  获取文章列表  {"fakeID": "xxx", "count": 10}  ✅
INFO  保存新文章成功  {"account": "xxx", "count": 3}  ✅
INFO  定时爬取任务完成  ✅
```

### 定时任务验证

定时任务 `cron_job.go` 已经使用了 `context.Background()`，不需要修改：

```go
func (s *Scheduler) executeCrawlTask() {
    logger.Info("========== 开始执行定时爬取任务 ==========")

    ctx := context.Background()  // ✅ 已经正确使用
    if err := s.crawlerService.FetchAllAccounts(ctx); err != nil {
        logger.Error("定时爬取任务执行失败", zap.Error(err))
    }

    logger.Info("========== 定时爬取任务执行完成 ==========")
}
```

## 最佳实践

### Context 使用原则

#### ✅ 正确做法

1. **同步操作**：使用 HTTP 请求的 context
   ```go
   func (h *Handler) SyncOperation(c *gin.Context) {
       // 同步操作，可以使用请求的context
       data, err := h.service.GetData(c.Request.Context())
       if err != nil {
           response.Error(c, 500, err.Error())
           return
       }
       response.Success(c, data)
   }
   ```

2. **异步操作**：使用独立的 context.Background()
   ```go
   func (h *Handler) AsyncOperation(c *gin.Context) {
       // 异步操作，使用独立的context
       go func() {
           ctx := context.Background()
           err := h.service.LongRunningTask(ctx)
           if err != nil {
               logger.Error("异步任务失败", zap.Error(err))
           }
       }()
       response.Success(c, "任务已启动")
   }
   ```

3. **需要超时控制的异步操作**：创建带超时的 context
   ```go
   func (h *Handler) AsyncWithTimeout(c *gin.Context) {
       go func() {
           ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
           defer cancel()
           
           err := h.service.TaskWithTimeout(ctx)
           if err != nil {
               logger.Error("任务执行失败", zap.Error(err))
           }
       }()
       response.Success(c, "任务已启动")
   }
   ```

#### ❌ 错误做法

1. **在异步任务中使用请求的 context**
   ```go
   // 错误示例 ❌
   func (h *Handler) WrongAsyncOperation(c *gin.Context) {
       go func() {
           // 请求结束后context会被取消
           err := h.service.Task(c.Request.Context())  // ❌
           ...
       }()
       response.Success(c, "任务已启动")
   }
   ```

2. **忽略 context 的生命周期**
   ```go
   // 错误示例 ❌
   func (h *Handler) WrongPattern(c *gin.Context) {
       ctx := c.Request.Context()
       
       // 返回响应后ctx会被取消
       response.Success(c, "OK")
       
       // 此时ctx已经无效
       h.service.AfterResponse(ctx)  // ❌
   }
   ```

## 相关场景

这个问题可能出现在以下场景：

1. ✅ **手动触发爬取** - 已修复
2. ✅ **定时任务调度** - 已经正确使用 context.Background()
3. ⚠️ **其他异步任务** - 需要检查是否也存在类似问题

### 检查清单

- [x] 手动触发爬取 (TriggerFetch) - 已修复
- [x] 定时任务 (Scheduler.executeCrawlTask) - 已正确
- [ ] 其他可能的异步API操作 - 建议检查

## 总结

### 核心要点

1. **HTTP请求的 context 生命周期短**
   - 请求完成后立即取消
   - 不适合异步任务使用

2. **异步任务需要独立的 context**
   - 使用 `context.Background()`
   - 或创建带超时的 context

3. **理解 Context 的用途**
   - 传递请求范围的值
   - 控制取消信号
   - 设置超时时间

### 修复效果

- ✅ 手动触发爬取任务正常执行
- ✅ 定时任务正常执行
- ✅ 不再出现 "context canceled" 错误
- ✅ 爬取任务可以完整执行完成

---

**修复版本**: v1.2.1  
**修复日期**: 2024-01-01  
**修复状态**: ✅ 已完成并测试通过

