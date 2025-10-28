package service

import (
	"context"
	"fmt"
	"sync"

	"wechat-crawler/internal/crawler"
	"wechat-crawler/internal/model"
	"wechat-crawler/internal/repository"
	"wechat-crawler/pkg/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// CrawlerService 爬虫业务逻辑服务
type CrawlerService struct {
	browser     *crawler.Browser
	wechatRepo  *repository.WeChatAccountRepo
	articleRepo *repository.ArticleRepo
	concurrent  int
	fetchCount  int
	mu          sync.Mutex
}

// NewCrawlerService 创建爬虫服务实例
func NewCrawlerService(browser *crawler.Browser, concurrent int) *CrawlerService {
	return &CrawlerService{
		browser:     browser,
		wechatRepo:  repository.NewWeChatAccountRepo(),
		articleRepo: repository.NewArticleRepo(),
		concurrent:  concurrent,
		fetchCount:  10, // 每次获取最新10篇文章
	}
}

// AddAccount 添加公众号订阅
func (s *CrawlerService) AddAccount(ctx context.Context, name, alias string) (*model.WeChatAccount, error) {
	logger.Info("添加公众号订阅", zap.String("name", name), zap.String("alias", alias))

	// 检查是否已存在
	existingAccount, err := s.wechatRepo.FindByName(ctx, name)
	if err == nil && existingAccount != nil {
		return nil, fmt.Errorf("公众号已存在")
	}

	// 搜索公众号获取FakeID
	fakeID, err := s.browser.SearchAccount(name)
	if err != nil {
		return nil, fmt.Errorf("搜索公众号失败: %w", err)
	}

	// 创建公众号记录
	account := &model.WeChatAccount{
		Name:   name,
		Alias:  alias,
		FakeID: fakeID,
		Status: 1,
	}

	if err := s.wechatRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("保存公众号失败: %w", err)
	}

	logger.Info("添加公众号成功",
		zap.String("name", name),
		zap.String("fakeID", fakeID),
		zap.String("id", account.ID.Hex()))

	return account, nil
}

// GetAccountList 获取公众号列表
func (s *CrawlerService) GetAccountList(ctx context.Context) ([]*model.WeChatAccount, error) {
	return s.wechatRepo.List(ctx)
}

// DeleteAccount 删除公众号订阅
func (s *CrawlerService) DeleteAccount(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("无效的ID")
	}

	return s.wechatRepo.Delete(ctx, objectID)
}

// FetchLatestArticles 获取指定公众号的最新文章
func (s *CrawlerService) FetchLatestArticles(ctx context.Context, account *model.WeChatAccount) ([]*model.Article, error) {
	logger.Info("开始获取公众号文章",
		zap.String("name", account.Name),
		zap.String("fakeID", account.FakeID))

	// 获取文章列表
	articleList, err := s.browser.FetchArticles(account.FakeID, s.fetchCount)
	if err != nil {
		return nil, fmt.Errorf("获取文章列表失败: %w", err)
	}

	if len(articleList) == 0 {
		logger.Info("没有找到文章", zap.String("account", account.Name))
		return nil, nil
	}

	// 检查是否有新文章
	var newArticles []*model.Article
	for _, item := range articleList {
		// 如果文章URL等于last_article，说明之前的文章都已采集过
		if account.LastArticle != "" && item.ContentURL == account.LastArticle {
			logger.Info("已到达上次采集位置", zap.String("url", item.ContentURL))
			break
		}

		// 检查文章是否已存在
		exists, err := s.articleRepo.ExistsByContentURL(ctx, item.ContentURL)
		if err != nil {
			logger.Warn("检查文章是否存在失败", zap.Error(err))
			continue
		}

		if exists {
			logger.Debug("文章已存在，跳过", zap.String("title", item.Title))
			continue
		}

		// 获取文章详细内容
		content, err := s.browser.FetchArticleContent(item.ContentURL)
		if err != nil {
			logger.Warn("获取文章内容失败", zap.String("url", item.ContentURL), zap.Error(err))
			content = "" // 即使获取内容失败也保存文章元数据
		}

		// 构造文章对象
		article := &model.Article{
			AccountID:   account.ID,
			AccountName: account.Name,
			Title:       item.Title,
			Author:      item.Author,
			Digest:      item.Digest,
			Content:     content,
			ContentURL:  item.ContentURL,
			Cover:       item.Cover,
			SourceURL:   item.SourceURL,
			PublishTime: item.CreateTime,
		}

		newArticles = append(newArticles, article)
	}

	// 批量保存新文章
	if len(newArticles) > 0 {
		if err := s.articleRepo.BatchCreate(ctx, newArticles); err != nil {
			return nil, fmt.Errorf("保存文章失败: %w", err)
		}

		// 更新公众号的最后文章URL
		latestArticleURL := newArticles[0].ContentURL
		if err := s.wechatRepo.UpdateLastArticle(ctx, account.ID, latestArticleURL); err != nil {
			logger.Warn("更新最后文章URL失败", zap.Error(err))
		}

		logger.Info("保存新文章成功",
			zap.String("account", account.Name),
			zap.Int("count", len(newArticles)))
	} else {
		logger.Info("没有新文章", zap.String("account", account.Name))
	}

	return newArticles, nil
}

// FetchAllAccounts 爬取所有订阅的公众号
func (s *CrawlerService) FetchAllAccounts(ctx context.Context) error {
	logger.Info("开始执行定时爬取任务")

	// 获取所有公众号
	accounts, err := s.wechatRepo.List(ctx)
	if err != nil {
		logger.Error("获取公众号列表失败", zap.Error(err))
		return err
	}

	if len(accounts) == 0 {
		logger.Info("没有订阅的公众号")
		return nil
	}

	logger.Info("待爬取公众号数量", zap.Int("count", len(accounts)))

	// 使用goroutine和channel控制并发爬取
	semaphore := make(chan struct{}, s.concurrent)
	var wg sync.WaitGroup

	for _, account := range accounts {
		wg.Add(1)
		go func(acc *model.WeChatAccount) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 执行爬取
			articles, err := s.FetchLatestArticles(ctx, acc)
			if err != nil {
				logger.Error("爬取公众号失败",
					zap.String("account", acc.Name),
					zap.Error(err))
				return
			}

			if len(articles) > 0 {
				logger.Info("发现新文章",
					zap.String("account", acc.Name),
					zap.Int("count", len(articles)))
			}
		}(account)
	}

	wg.Wait()
	logger.Info("定时爬取任务完成")
	return nil
}

// GetArticleList 获取文章列表
func (s *CrawlerService) GetArticleList(ctx context.Context, accountID string, page, pageSize int64) ([]*model.Article, int64, error) {
	if accountID != "" {
		// 获取指定公众号的文章
		objectID, err := primitive.ObjectIDFromHex(accountID)
		if err != nil {
			return nil, 0, fmt.Errorf("无效的公众号ID")
		}
		return s.articleRepo.ListByAccountID(ctx, objectID, page, pageSize)
	}

	// 获取所有文章
	return s.articleRepo.List(ctx, page, pageSize)
}

// GetAccount 获取公众号详情
func (s *CrawlerService) GetAccount(ctx context.Context, id string) (*model.WeChatAccount, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("无效的ID")
	}

	account, err := s.wechatRepo.FindByID(ctx, objectID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("公众号不存在")
		}
		return nil, err
	}

	return account, nil
}
