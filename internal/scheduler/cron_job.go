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
	interval       int // 爬取间隔（分钟）
}

// NewScheduler 创建调度器实例
func NewScheduler(crawlerService *service.CrawlerService, interval int) *Scheduler {
	return &Scheduler{
		cron:           cron.New(cron.WithSeconds()),
		crawlerService: crawlerService,
		interval:       interval,
	}
}

// Start 启动定时任务
func (s *Scheduler) Start() error {
	// 构建cron表达式：每N分钟执行一次
	// 格式：秒 分 时 日 月 周
	cronExpr := fmt.Sprintf("0 */%d * * * *", s.interval)

	logger.Info("配置定时任务",
		zap.Int("interval_minutes", s.interval),
		zap.String("cron_expr", cronExpr))

	// 添加定时任务
	_, err := s.cron.AddFunc(cronExpr, func() {
		s.executeCrawlTask()
	})

	if err != nil {
		logger.Error("添加定时任务失败", zap.Error(err))
		return err
	}

	// 启动调度器
	s.cron.Start()
	logger.Info("定时任务调度器已启动")

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

// RunOnce 立即执行一次爬取任务（用于测试或手动触发）
func (s *Scheduler) RunOnce() {
	logger.Info("手动触发爬取任务")
	s.executeCrawlTask()
}
