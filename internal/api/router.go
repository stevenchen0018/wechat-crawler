package api

import (
	"wechat-crawler/internal/api/handler"
	"wechat-crawler/internal/service"

	"github.com/gin-gonic/gin"
)

// SetupRouter 配置路由
func SetupRouter(crawlerService *service.CrawlerService) *gin.Engine {
	// 设置Gin模式
	gin.SetMode(gin.DebugMode)

	r := gin.Default()

	// 创建处理器
	wechatHandler := handler.NewWeChatHandler(crawlerService)

	// API路由组
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

	return r
}
