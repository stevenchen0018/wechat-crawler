# 微信公众号订阅爬虫系统

基于 Go + Gin + MongoDB + chromedp 开发的微信公众号自动采集系统。

## 功能特性

### 核心功能
- 🔖 **公众号订阅管理** - 支持添加、查看、删除订阅的公众号
- 🤖 **自动文章采集** - 使用无头浏览器自动爬取公众号最新文章
- ⏰ **定时任务调度** - 自定义爬取间隔，自动检测所有订阅公众号的新文章
- 🍪 **Cookie复用机制** - 首次扫码登录后自动保存，避免重复扫码
- 📊 **结构化存储** - MongoDB持久化公众号信息和文章内容
- 📝 **完善日志系统** - 使用zap记录所有操作和错误信息

### 管理后台
- 🎨 **现代化界面** - 基于Bootstrap 5的响应式管理后台
- 🔐 **安全认证** - 账户密码登录 + 图形验证码保护
- 📈 **数据统计** - 实时展示订阅数、文章数等统计信息
- 📋 **列表管理** - 公众号列表、文章列表，支持搜索和筛选
- 🎮 **手动控制** - 支持手动触发爬取任务
- ⚙️ **系统设置** - 在线修改定时器间隔等配置项

### 技术优化
- 🔒 **并发控制保护** - 浏览器操作串行执行，防止微信平台封控
- 🛡️ **异常处理增强** - 自动识别已删除文章、超时、元素缺失等异常情况
- 🧹 **资源管理完善** - 程序退出时自动关闭浏览器进程

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
├── templates/                     # HTML模板
│   └── admin/
│       ├── layout.html           # 布局模板
│       ├── login.html            # 登录页面
│       ├── dashboard.html        # 仪表板
│       ├── accounts.html         # 公众号管理
│       ├── articles.html         # 文章管理
│       ├── tasks.html            # 任务管理
│       └── settings.html         # 系统设置
├── static/                        # 静态资源
│   ├── css/
│   │   └── admin.css            # 管理后台样式
│   └── js/
│       └── admin.js             # 管理后台脚本
├── internal/
│   ├── api/
│   │   ├── router.go            # 路由配置
│   │   └── handler/
│   │       ├── wechat_handler.go # API接口处理器
│   │       └── admin_handler.go  # 管理后台处理器
│   ├── middleware/
│   │   └── auth.go              # 认证中间件
│   ├── model/
│   │   ├── user.go              # 用户模型
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
│   ├── session/
│   │   └── session.go          # Session管理
│   ├── captcha/
│   │   └── captcha.go          # 验证码生成
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
go mod tidy
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
4. 访问 `http://localhost:8081/admin` 进入管理后台
5. 默认账号：`admin`，密码：`admin123`

## 管理后台使用

### 登录管理后台

访问 `http://localhost:8081/admin` 进入登录页面，输入用户名、密码和验证码即可登录。

默认账号信息：
- 用户名：`admin`
- 密码：`admin123`

> 💡 **安全提示**：首次登录后，请修改配置文件中的密码。密码采用bcrypt加密，可以使用在线工具生成。

### 主要功能页面

1. **仪表板** - 查看系统概览和统计信息
2. **公众号管理** - 添加/删除订阅，查看公众号列表
3. **文章管理** - 查看采集的文章，支持按公众号筛选
4. **任务管理** - 查看定时任务状态，手动触发爬取
5. **系统设置** - 修改爬取间隔等配置项

### 修改管理员密码

1. 使用bcrypt工具生成密码哈希：

```bash
# 在线工具：https://bcrypt-generator.com/
# 或使用Go代码生成
```

2. 修改 `config/config.yaml` 中的 `admin.password` 字段
3. 重启服务使配置生效

## API 接口

### 公众号管理 API

#### 1. 添加公众号订阅

```http
POST /api/wechat/add
Content-Type: application/json

{
  "name": "公众号名称",
  "alias": "公众号别名（可选）"
}
```

#### 2. 查看已订阅公众号

```http
GET /api/wechat/list
```

#### 3. 取消订阅公众号

```http
DELETE /api/wechat/:id
```

#### 4. 查看文章列表

```http
GET /api/article/list?account_id=xxx&page=1&page_size=20
```

### 管理后台 API（需要登录）

#### 1. 手动触发爬取

```http
POST /admin/api/tasks/trigger
```

#### 2. 更新系统设置

```http
POST /admin/api/settings/update
Content-Type: application/json

{
  "crawl_interval": 60,  // 爬取间隔（分钟）
  "fetch_count": 10,     // 每次获取文章数量
  "timeout": 60          // 超时时间（秒）
}
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
5. ⚠️ 系统已实现并发控制，所有浏览器操作串行执行以避免封控
6. ⚠️ 已删除或不可访问的文章会被自动跳过，不会保存到数据库

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

