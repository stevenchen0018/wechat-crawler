package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wechat-crawler/internal/api"
	"wechat-crawler/internal/crawler"
	"wechat-crawler/internal/scheduler"
	"wechat-crawler/internal/service"
	"wechat-crawler/pkg/database"
	"wechat-crawler/pkg/logger"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	if err := loadConfig(); err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	if err := logger.Init(
		viper.GetString("log.output"),
		viper.GetString("log.level"),
	); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("========== 微信公众号爬虫系统启动 ==========")

	// 初始化MongoDB
	if err := database.Init(database.Config{
		URI:      viper.GetString("mongodb.uri"),
		Database: viper.GetString("mongodb.database"),
		Timeout:  viper.GetInt("mongodb.timeout"),
	}); err != nil {
		logger.Fatal("初始化MongoDB失败", zap.Error(err))
	}
	defer database.Close()
	logger.Info("MongoDB连接成功")

	// 创建浏览器实例
	browser, err := crawler.NewBrowser(
		viper.GetString("crawler.cookie_file"),
		viper.GetString("wechat.mp_url"),
		viper.GetInt("crawler.timeout"),
		viper.GetBool("crawler.debug_mode"), // 从配置文件读取debug模式
	)
	if err != nil {
		logger.Fatal("创建浏览器实例失败", zap.Error(err))
	}
	defer browser.Close()
	logger.Info("浏览器实例创建成功")

	// 执行登录
	// if err := browser.Login(); err != nil {
	// 	logger.Fatal("登录微信公众号平台失败", zap.Error(err))
	// }
	// logger.Info("微信公众号平台登录成功")

	// 创建爬虫服务
	crawlerService := service.NewCrawlerService(
		browser,
		viper.GetInt("crawler.concurrent"),
	)

	// 创建飞书服务
	feishuService := service.NewFeishuService()

	// 启动定时任务
	cronScheduler := scheduler.NewScheduler(
		crawlerService,
		feishuService,
		viper.GetInt("crawler.interval"),
	)
	if err := cronScheduler.Start(); err != nil {
		logger.Fatal("启动定时任务失败", zap.Error(err))
	}
	defer cronScheduler.Stop()

	// 设置路由并启动HTTP服务
	router := api.SetupRouter(crawlerService)

	// 获取服务端口
	port := viper.GetString("server.port")
	serverAddr := fmt.Sprintf(":%s", port)

	// 启动HTTP服务（在goroutine中运行）
	go func() {
		logger.Info("HTTP服务启动", zap.String("address", serverAddr))
		if err := router.Run(serverAddr); err != nil {
			logger.Fatal("HTTP服务启动失败", zap.Error(err))
		}
	}()

	// 等待中断信号优雅关闭服务
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("收到退出信号，开始优雅关闭...")

	// 设置关闭超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 等待正在执行的任务完成
	select {
	case <-ctx.Done():
		logger.Warn("关闭超时，强制退出")
	default:
		logger.Info("服务已优雅关闭")
	}

	// Debug模式：提示用户并等待
	if viper.GetBool("crawler.debug_mode") {
		logger.Info("========================================")
		logger.Info("🔍 Debug模式：浏览器将保持打开状态")
		logger.Info("📌 请在浏览器中完成调试操作")
		logger.Info("⚠️  按 Ctrl+C 可以退出程序并关闭浏览器")
		logger.Info("========================================")

		// 创建新的信号通道，等待第二次退出信号
		quit2 := make(chan os.Signal, 1)
		signal.Notify(quit2, syscall.SIGINT, syscall.SIGTERM)
		<-quit2

		logger.Info("收到第二次退出信号，关闭浏览器...")
	}

	logger.Info("========== 微信公众号爬虫系统已退出 ==========")
}

// loadConfig 加载配置文件
func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("../config")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	viper.SetDefault("mongodb.database", "wechat_crawler")
	viper.SetDefault("mongodb.timeout", 10)
	viper.SetDefault("crawler.interval", 10)
	viper.SetDefault("crawler.concurrent", 3)
	viper.SetDefault("crawler.timeout", 60)
	viper.SetDefault("crawler.cookie_file", "./cookie.json")
	viper.SetDefault("crawler.debug_mode", false)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "./logs/app.log")
	viper.SetDefault("wechat.mp_url", "https://mp.weixin.qq.com")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("配置文件不存在，使用默认配置")
			return nil
		}
		return err
	}

	fmt.Println("配置文件加载成功:", viper.ConfigFileUsed())
	return nil
}
