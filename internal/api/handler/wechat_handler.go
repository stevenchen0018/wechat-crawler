package handler

import (
	"context"
	"strconv"

	"wechat-crawler/internal/service"
	"wechat-crawler/pkg/logger"
	"wechat-crawler/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WeChatHandler 微信公众号处理器
type WeChatHandler struct {
	crawlerService *service.CrawlerService
}

// NewWeChatHandler 创建处理器实例
func NewWeChatHandler(crawlerService *service.CrawlerService) *WeChatHandler {
	return &WeChatHandler{
		crawlerService: crawlerService,
	}
}

// AddAccountRequest 添加公众号请求
type AddAccountRequest struct {
	Name  string `json:"name" binding:"required"`
	Alias string `json:"alias"`
}

// AddAccount 添加公众号订阅
// @Summary 添加公众号订阅
// @Description 添加一个新的微信公众号订阅
// @Tags 公众号管理
// @Accept json
// @Produce json
// @Param body body AddAccountRequest true "公众号信息"
// @Success 200 {object} response.Response
// @Router /api/wechat/add [post]
func (h *WeChatHandler) AddAccount(c *gin.Context) {
	var req AddAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("请求参数错误", zap.Error(err))
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	account, err := h.crawlerService.AddAccount(c.Request.Context(), req.Name, req.Alias)
	if err != nil {
		logger.Error("添加公众号失败", zap.Error(err))
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, account)
}

// GetAccountList 获取公众号列表
// @Summary 获取公众号列表
// @Description 获取所有已订阅的公众号列表
// @Tags 公众号管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/wechat/list [get]
func (h *WeChatHandler) GetAccountList(c *gin.Context) {
	accounts, err := h.crawlerService.GetAccountList(c.Request.Context())
	if err != nil {
		logger.Error("获取公众号列表失败", zap.Error(err))
		response.InternalServerError(c, "获取列表失败")
		return
	}

	response.Success(c, accounts)
}

// GetAccount 获取公众号详情
// @Summary 获取公众号详情
// @Description 根据ID获取公众号详细信息
// @Tags 公众号管理
// @Produce json
// @Param id path string true "公众号ID"
// @Success 200 {object} response.Response
// @Router /api/wechat/:id [get]
func (h *WeChatHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "ID不能为空")
		return
	}

	account, err := h.crawlerService.GetAccount(c.Request.Context(), id)
	if err != nil {
		logger.Error("获取公众号详情失败", zap.String("id", id), zap.Error(err))
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, account)
}

// DeleteAccount 删除公众号订阅
// @Summary 删除公众号订阅
// @Description 取消订阅指定的公众号
// @Tags 公众号管理
// @Produce json
// @Param id path string true "公众号ID"
// @Success 200 {object} response.Response
// @Router /api/wechat/:id [delete]
func (h *WeChatHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "ID不能为空")
		return
	}

	err := h.crawlerService.DeleteAccount(c.Request.Context(), id)
	if err != nil {
		logger.Error("删除公众号失败", zap.String("id", id), zap.Error(err))
		response.InternalServerError(c, "删除失败")
		return
	}

	response.SuccessWithMsg(c, "删除成功", nil)
}

// GetArticleList 获取文章列表
// @Summary 获取文章列表
// @Description 获取文章列表，支持分页和按公众号筛选
// @Tags 文章管理
// @Produce json
// @Param account_id query string false "公众号ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response
// @Router /api/article/list [get]
func (h *WeChatHandler) GetArticleList(c *gin.Context) {
	accountID := c.Query("account_id")

	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 64)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	articles, total, err := h.crawlerService.GetArticleList(c.Request.Context(), accountID, page, pageSize)
	if err != nil {
		logger.Error("获取文章列表失败", zap.Error(err))
		response.InternalServerError(c, "获取列表失败")
		return
	}

	response.SuccessWithPage(c, articles, total, page, pageSize)
}

// TriggerFetch 手动触发爬取任务
// @Summary 手动触发爬取
// @Description 立即执行一次所有公众号的爬取任务
// @Tags 爬取任务
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/crawler/trigger [post]
func (h *WeChatHandler) TriggerFetch(c *gin.Context) {
	logger.Info("手动触发爬取任务")

	// 使用独立的context，不依赖HTTP请求的生命周期
	// 避免HTTP响应后context被取消导致爬取任务失败
	go func() {
		ctx := context.Background()
		if err := h.crawlerService.FetchAllAccounts(ctx); err != nil {
			logger.Error("爬取任务执行失败", zap.Error(err))
		}
	}()

	response.SuccessWithMsg(c, "爬取任务已启动", nil)
}
