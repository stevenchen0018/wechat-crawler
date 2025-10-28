# 微信公众号订阅爬虫系统

基于 Go + Gin + MongoDB + chromedp 开发的微信公众号自动采集系统。

## 功能特性

- 🔖 **公众号订阅管理** - 支持添加、查看、删除订阅的公众号
- 🤖 **自动文章采集** - 使用无头浏览器自动爬取公众号最新文章
- ⏰ **定时任务调度** - 每10分钟自动检测所有订阅公众号的新文章
- 🍪 **Cookie复用机制** - 首次扫码登录后自动保存，避免重复扫码
- 📊 **结构化存储** - MongoDB持久化公众号信息和文章内容
- 📝 **完善日志系统** - 使用zap记录所有操作和错误信息

## 技术栈

- **开发语言**: Go 1.22+
- **Web框架**: Gin
- **数据库**: MongoDB
- **无头浏览器**: chromedp
- **定时任务**: robfig/cron/v3
- **日志**: zap
- **配置管理**: viper

## 项目结构

```
wechat-crawler/
├── cmd/
│   └── main.go                    # 程序入口
├── config/
│   └── config.yaml               # 配置文件
├── internal/
│   ├── api/
│   │   ├── router.go            # 路由配置
│   │   └── handler/
│   │       └── wechat_handler.go # 公众号管理接口
│   ├── model/
│   │   ├── wechat_account.go   # 公众号数据模型
│   │   └── article.go          # 文章数据模型
│   ├── service/
│   │   └── crawler_service.go  # 爬虫业务逻辑
│   ├── crawler/
│   │   ├── browser.go          # chromedp浏览器封装
│   │   └── cookie.go           # Cookie管理
│   ├── repository/
│   │   ├── wechat_repo.go      # 公众号数据访问
│   │   └── article_repo.go     # 文章数据访问
│   └── scheduler/
│       └── cron_job.go         # 定时任务
├── pkg/
│   ├── logger/
│   │   └── logger.go           # 日志封装
│   ├── response/
│   │   └── response.go         # 统一响应格式
│   └── database/
│       └── mongodb.go          # MongoDB连接
├── go.mod
└── README.md
```

## 快速开始

### 前置要求

- Go 1.22+
- MongoDB 4.4+
- Chrome/Chromium 浏览器

### 安装依赖

```bash
go mod download
```

### 配置文件

编辑 `config/config.yaml` 配置 MongoDB 连接信息和其他参数。

### 运行程序

```bash
go run cmd/main.go
```

### 首次使用

1. 程序启动后会自动打开浏览器窗口
2. 使用微信扫码登录公众号后台（只需扫码一次）
3. 登录成功后 Cookie 会自动保存，后续无需再次扫码

## API 接口

### 1. 添加公众号订阅

```http
POST /api/wechat/add
Content-Type: application/json

{
  "name": "公众号名称",
  "alias": "公众号别名（可选）"
}
```

### 2. 查看已订阅公众号

```http
GET /api/wechat/list
```

### 3. 取消订阅公众号

```http
DELETE /api/wechat/:id
```

### 4. 查看文章列表

```http
GET /api/article/list?account_id=xxx&page=1&page_size=20
```

## 响应格式

所有接口返回统一的 JSON 格式：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

## 工作原理

1. **订阅管理**: 通过 API 添加需要订阅的公众号信息
2. **定时爬取**: 系统每10分钟自动执行一次爬取任务
3. **文章检测**: 访问微信公众号后台，获取最新文章列表
4. **去重判断**: 根据文章URL判断是否为新文章
5. **内容保存**: 新文章自动保存到 MongoDB，同时更新公众号的最后文章记录

## 注意事项

1. ⚠️ 首次运行需要手动扫码登录微信公众号平台
2. ⚠️ Cookie 有效期通常为几天到几周，过期后需要重新扫码
3. ⚠️ 请勿频繁爬取，建议爬取间隔不低于5分钟
4. ⚠️ 本项目仅供学习交流使用，请遵守相关法律法规

## 日志说明

日志文件默认保存在 `./logs/app.log`，包含：

- 系统启动/关闭信息
- 爬取任务执行情况
- 新文章发现记录
- 错误和异常信息

## License

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

