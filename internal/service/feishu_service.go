package service

import (
	"context"
	"fmt"
	"time"

	"wechat-crawler/internal/model"
	"wechat-crawler/internal/repository"
	"wechat-crawler/pkg/feishu"
	"wechat-crawler/pkg/logger"

	"go.uber.org/zap"
)

// FeishuService 飞书通知服务
type FeishuService struct {
	feishuRepo  *repository.FeishuConfigRepo
	articleRepo *repository.ArticleRepo
}

// NewFeishuService 创建飞书服务实例
func NewFeishuService() *FeishuService {
	return &FeishuService{
		feishuRepo:  repository.NewFeishuConfigRepo(),
		articleRepo: repository.NewArticleRepo(),
	}
}

// GetConfig 获取飞书配置
func (s *FeishuService) GetConfig(ctx context.Context) (*model.FeishuConfig, error) {
	return s.feishuRepo.GetConfig(ctx)
}

// SaveConfig 保存飞书配置
func (s *FeishuService) SaveConfig(ctx context.Context, config *model.FeishuConfig) error {
	return s.feishuRepo.UpsertConfig(ctx, config)
}

// TestNotification 测试飞书通知
func (s *FeishuService) TestNotification(ctx context.Context) error {
	config, err := s.feishuRepo.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("获取飞书配置失败: %w", err)
	}

	if config.WebhookURL == "" {
		return fmt.Errorf("飞书webhook地址未配置")
	}

	notifier := feishu.NewFeishuNotifier(config.WebhookURL)
	return notifier.TestNotification()
}

// SendArticleNotification 发送文章通知
func (s *FeishuService) SendArticleNotification(ctx context.Context) error {
	// 获取配置
	config, err := s.feishuRepo.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("获取飞书配置失败: %w", err)
	}

	if !config.Enabled {
		logger.Info("飞书通知未启用")
		return nil
	}

	if config.WebhookURL == "" {
		return fmt.Errorf("飞书webhook地址未配置")
	}

	// 获取最近的文章（根据通知周期）
	var startTime int64
	now := time.Now()

	switch config.NotifyPeriod {
	case "hourly":
		// 获取最近1小时的文章
		startTime = now.Add(-1 * time.Hour).Unix()
	case "daily":
		// 获取最近24小时的文章
		startTime = now.Add(-24 * time.Hour).Unix()
	default:
		// 默认24小时
		startTime = now.Add(-24 * time.Hour).Unix()
	}

	// 查询文章
	articles, _, err := s.articleRepo.ListWithFilter(ctx, "", "", startTime, 0, 1, 100)
	if err != nil {
		return fmt.Errorf("查询文章失败: %w", err)
	}

	if len(articles) == 0 {
		logger.Info("没有新文章，跳过飞书通知")
		return nil
	}

	// 发送通知
	notifier := feishu.NewFeishuNotifier(config.WebhookURL)
	title := config.NotifyTitle
	if title == "" {
		title = "微信公众号文章推送"
	}

	// 使用卡片消息发送
	if err := notifier.SendArticleCard(title, articles); err != nil {
		// 如果卡片消息发送失败，尝试使用文本消息
		logger.Warn("发送卡片消息失败，尝试使用文本消息", zap.Error(err))
		if err := notifier.SendArticleNotification(title, articles); err != nil {
			return fmt.Errorf("发送飞书通知失败: %w", err)
		}
	}

	logger.Info("飞书通知发送成功", zap.Int("article_count", len(articles)))
	return nil
}

