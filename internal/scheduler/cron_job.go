package scheduler

import (
	"context"
	"fmt"

	"wechat-crawler/internal/service"
	"wechat-crawler/pkg/logger"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	cron           *cron.Cron
	crawlerService *service.CrawlerService
	feishuService  *service.FeishuService
	interval       int // 爬取间隔（分钟）
}

// NewScheduler 创建调度器实例
func NewScheduler(crawlerService *service.CrawlerService, feishuService *service.FeishuService, interval int) *Scheduler {
	return &Scheduler{
		cron:           cron.New(cron.WithSeconds()),
		crawlerService: crawlerService,
		feishuService:  feishuService,
		interval:       interval,
	}
}

// Start 启动定时任务
func (s *Scheduler) Start() error {
	// 构建cron表达式：每N分钟执行一次
	// 格式：秒 分 时 日 月 周
	cronExpr, err := s.BuildCronExpr(s.interval)
	if err != nil {
		logger.Error("构建cron表达式失败", zap.Error(err))
		return err
	}
	// cronExpr := fmt.Sprintf("0 */%d * * * *", s.interval)

	logger.Info("配置爬取任务定时器",
		zap.Int("interval_minutes", s.interval),
		zap.String("cron_expr", cronExpr))

	// 添加爬取定时任务
	_, err = s.cron.AddFunc(cronExpr, func() {
		s.executeCrawlTask()
	})

	if err != nil {
		logger.Error("添加爬取定时任务失败", zap.Error(err))
		return err
	}

	// 添加飞书通知定时任务
	if err := s.setupFeishuNotifyTask(); err != nil {
		logger.Warn("配置飞书通知任务失败", zap.Error(err))
	}

	// 启动调度器
	s.cron.Start()
	logger.Info("定时任务调度器已启动")

	return nil
}

// setupFeishuNotifyTask 配置飞书通知定时任务
func (s *Scheduler) setupFeishuNotifyTask() error {
	ctx := context.Background()

	// 获取飞书配置
	config, err := s.feishuService.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("获取飞书配置失败: %w", err)
	}

	if !config.Enabled {
		logger.Info("飞书通知未启用，跳过配置定时任务")
		return nil
	}

	// 构建cron表达式
	var cronExpr string
	if config.NotifyPeriod == "hourly" {
		// 每小时执行一次
		cronExpr = "0 0 * * * *"
	} else {
		// 每天在指定时间执行
		// 解析时间字符串 "HH:MM"
		if config.NotifyTime == "" {
			config.NotifyTime = "09:00"
		}
		// 格式：秒 分 时 日 月 周
		cronExpr = fmt.Sprintf("0 %s * * *", config.NotifyTime)
	}

	logger.Info("配置飞书通知定时器",
		zap.String("period", config.NotifyPeriod),
		zap.String("time", config.NotifyTime),
		zap.String("cron_expr", cronExpr))

	// 添加定时任务
	_, err = s.cron.AddFunc(cronExpr, func() {
		s.executeFeishuNotifyTask()
	})

	if err != nil {
		return fmt.Errorf("添加飞书通知定时任务失败: %w", err)
	}

	return nil
}

// Stop 停止定时任务
func (s *Scheduler) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
		logger.Info("定时任务调度器已停止")
	}
}

// executeCrawlTask 执行爬取任务
func (s *Scheduler) executeCrawlTask() {
	logger.Info("========== 开始执行定时爬取任务 ==========")

	ctx := context.Background()
	if err := s.crawlerService.FetchAllAccounts(ctx); err != nil {
		logger.Error("定时爬取任务执行失败", zap.Error(err))
	}

	logger.Info("========== 定时爬取任务执行完成 ==========")
}

// executeFeishuNotifyTask 执行飞书通知任务
func (s *Scheduler) executeFeishuNotifyTask() {
	logger.Info("========== 开始执行飞书通知任务 ==========")

	ctx := context.Background()
	if err := s.feishuService.SendArticleNotification(ctx); err != nil {
		logger.Error("飞书通知任务执行失败", zap.Error(err))
	}

	logger.Info("========== 飞书通知任务执行完成 ==========")
}

// RunOnce 立即执行一次爬取任务（用于测试或手动触发）
func (s *Scheduler) RunOnce() {
	logger.Info("手动触发爬取任务")
	s.executeCrawlTask()
}

// ReloadFeishuTask 重新加载飞书通知任务（配置更新后调用）
func (s *Scheduler) ReloadFeishuTask() error {
	logger.Info("重新加载飞书通知定时任务")
	return s.setupFeishuNotifyTask()
}

func (s *Scheduler) BuildCronExpr(interval int) (string, error) {
	if interval < 5 || interval > 1440 {
		return "", fmt.Errorf("interval must be between 5 and 1440 minutes")
	}

	// 小于 60 分钟：直接使用 */N
	if interval < 60 {
		return fmt.Sprintf("0 */%d * * * *", interval), nil
	}

	// 大于或等于 60 分钟
	hours := interval / 60   // 每 N 小时执行
	minutes := interval % 60 // 偏移分钟，例如 90 分钟 = 1 小时 + 30 分钟

	if minutes == 0 {
		// 整小时：每 N 小时执行一次
		return fmt.Sprintf("0 0 */%d * * *", hours), nil
	}

	// 带分钟偏移：例如 90 分钟 = 每1小时的第 30 分钟执行
	// 但要确保 interval 最终是 N 小时 + offset，要让策略逻辑能接受
	return fmt.Sprintf("0 %d */%d * * *", minutes, hours), nil
}
