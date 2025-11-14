package api

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wechat-crawler/internal/api/handler"
	"wechat-crawler/internal/middleware"
	"wechat-crawler/internal/service"
	"wechat-crawler/pkg/logger"
	"wechat-crawler/pkg/session"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// SetupRouter 配置路由
func SetupRouter(crawlerService *service.CrawlerService) *gin.Engine {
	// 设置Gin模式
	mode := viper.GetString("server.mode")
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()

	r.SetFuncMap(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"formatTime": func(ts int64) string {
			return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
		},
	})

	// 获取项目根目录
	rootDir := getProjectRoot()

	// 加载HTML模板
	templatesPath := filepath.Join(rootDir, "templates", "*.html")
	r.LoadHTMLGlob(templatesPath)

	// 静态文件服务
	staticPath := filepath.Join(rootDir, "static")
	r.Static("/static", staticPath)

	// 设置模板函数
	// r.SetFuncMap(template.FuncMap{
	// 	"sub": func(a, b int64) int64 {
	// 		return a - b
	// 	},
	// 	"add": func(a, b int64) int64 {
	// 		return a + b
	// 	},
	// })

	// // 获取项目根目录
	// rootDir := getProjectRoot()

	// // 构建静态文件的绝对路径
	// staticPath := filepath.Join(rootDir, "static")

	// // 加载HTML模板（使用自定义方式保留目录结构）
	// templatesDir := filepath.Join(rootDir, "templates")
	// if err := loadTemplatesWithDir(r, templatesDir); err != nil {
	// 	logger.Error("加载模板失败", zap.Error(err))
	// 	panic(err)
	// }

	// // 静态文件服务
	// r.Static("/static", staticPath)

	// 初始化会话存储（24小时过期）
	sessionStore := session.NewStore(24 * time.Hour)
	middleware.InitSession(sessionStore)

	// 创建处理器
	wechatHandler := handler.NewWeChatHandler(crawlerService)
	adminHandler := handler.NewAdminHandler(crawlerService, sessionStore)

	// 管理后台路由
	admin := r.Group("/admin")
	{
		// 登录相关（无需认证）
		admin.GET("/login", adminHandler.ShowLoginPage)
		admin.POST("/login", adminHandler.Login)
		admin.GET("/captcha", adminHandler.GetCaptcha)

		// 需要认证的页面
		adminAuth := admin.Group("")
		adminAuth.Use(middleware.AuthRequired())
		{
			adminAuth.GET("", adminHandler.ShowDashboard)         // 仪表板
			adminAuth.GET("/", adminHandler.ShowDashboard)        // 仪表板
			adminAuth.GET("/accounts", adminHandler.ShowAccounts) // 公众号管理
			adminAuth.GET("/articles", adminHandler.ShowArticles) // 文章管理
			adminAuth.GET("/tasks", adminHandler.ShowTasks)       // 任务管理
			adminAuth.GET("/settings", adminHandler.ShowSettings) // 系统设置
			adminAuth.GET("/logout", adminHandler.Logout)         // 退出登录
		}

		// 管理后台API（需要认证）
		adminAPI := admin.Group("/api")
		adminAPI.Use(middleware.AuthRequired())
		{
			adminAPI.POST("/tasks/trigger", adminHandler.TriggerCrawl)     // 手动触发爬取
			adminAPI.POST("/settings/update", adminHandler.UpdateSettings) // 更新设置
			adminAPI.GET("/logs", adminHandler.GetLogs)                    // 获取日志
		}
	}

	// 公开API路由组
	api := r.Group("/api")
	{
		// 公众号管理
		wechat := api.Group("/wechat")
		{
			wechat.POST("/add", wechatHandler.AddAccount)      // 添加公众号
			wechat.GET("/list", wechatHandler.GetAccountList)  // 获取公众号列表
			wechat.GET("/:id", wechatHandler.GetAccount)       // 获取公众号详情
			wechat.DELETE("/:id", wechatHandler.DeleteAccount) // 删除公众号
		}

		// 文章管理
		article := api.Group("/article")
		{
			article.GET("/list", wechatHandler.GetArticleList) // 获取文章列表
		}

		// 爬虫任务
		crawler := api.Group("/crawler")
		{
			crawler.POST("/trigger", wechatHandler.TriggerFetch) // 手动触发爬取
		}
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"msg":    "wechat-crawler service is running",
		})
	})

	// 根路径重定向到管理后台
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/admin")
	})

	return r
}

// loadTemplatesWithDir 加载模板并保留目录结构
func loadTemplatesWithDir(r *gin.Engine, templatesDir string) error {
	templ := template.New("")

	// 添加自定义函数
	templ.Funcs(template.FuncMap{
		"sub": func(a, b int64) int64 {
			return a - b
		},
		"add": func(a, b int64) int64 {
			return a + b
		},
	})

	// 遍历templates目录下的所有html文件
	err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			// 获取相对于templates目录的路径作为模板名称
			relPath, err := filepath.Rel(templatesDir, path)
			if err != nil {
				return err
			}

			// 将路径分隔符统一为/（Windows兼容）
			templateName := filepath.ToSlash(relPath)

			// 读取并解析模板
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// 使用模板名称解析
			_, err = templ.New(templateName).Parse(string(content))
			if err != nil {
				logger.Error("解析模板失败",
					zap.String("file", path),
					zap.String("name", templateName),
					zap.Error(err))
				return err
			}

			logger.Debug("加载模板", zap.String("name", templateName))
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 设置模板到Gin
	r.SetHTMLTemplate(templ)
	return nil
}

// getProjectRoot 获取项目根目录（包含go.mod的目录）
func getProjectRoot() string {
	// 尝试从当前工作目录开始查找
	currentDir, err := os.Getwd()
	if err != nil {
		// 如果获取失败，尝试使用可执行文件所在目录
		if execPath, err := os.Executable(); err == nil {
			currentDir = filepath.Dir(execPath)
		} else {
			return "."
		}
	}

	// 向上查找包含 go.mod 的目录
	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// 找到项目根目录
			return currentDir
		}

		// 向上一级目录
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// 已经到达文件系统根目录，停止查找
			break
		}
		currentDir = parentDir
	}

	// 如果找不到，返回当前工作目录
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}
