package handler

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"wechat-crawler/internal/middleware"
	"wechat-crawler/internal/model"
	"wechat-crawler/internal/service"
	"wechat-crawler/pkg/captcha"
	"wechat-crawler/pkg/logger"
	"wechat-crawler/pkg/response"
	"wechat-crawler/pkg/session"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AdminHandler 管理后台处理器
type AdminHandler struct {
	crawlerService *service.CrawlerService
	sessionStore   *session.Store
}

// NewAdminHandler 创建管理后台处理器
func NewAdminHandler(crawlerService *service.CrawlerService, sessionStore *session.Store) *AdminHandler {
	return &AdminHandler{
		crawlerService: crawlerService,
		sessionStore:   sessionStore,
	}
}

// ShowLoginPage 显示登录页面
func (h *AdminHandler) ShowLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"Title":   "登录",
		"IsLogin": true,
	})
}

// Login 处理登录
func (h *AdminHandler) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	captchaAnswer := c.PostForm("captcha")
	captchaID := c.PostForm("captcha_id")

	// 从Cookie获取验证码ID
	captchaID, _ = c.Cookie("captcha_id")

	// 验证验证码
	if !captcha.Verify(captchaID, captchaAnswer) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"Title":   "登录",
			"IsLogin": false,
			"Error":   "验证码错误",
		})
		return
	}

	// 验证用户名和密码（这里使用配置文件中的账号密码）
	configUsername := viper.GetString("admin.username")
	configPassword := viper.GetString("admin.password")

	if username != configUsername {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"Title":   "登录",
			"IsLogin": false,
			"Error":   "用户名或密码错误",
		})
		return
	}

	// 验证密码（bcrypt）
	if err := bcrypt.CompareHashAndPassword([]byte(configPassword), []byte(password)); err != nil {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"Title":   "登录",
			"IsLogin": false,
			"Error":   "用户名或密码错误",
		})
		return
	}

	// 创建会话
	sess, err := h.sessionStore.Create()
	if err != nil {
		logger.Error("创建会话失败", zap.Error(err))
		c.HTML(http.StatusOK, "login.html", gin.H{
			"Title":   "登录",
			"IsLogin": false,
			"Error":   "登录失败，请重试",
		})
		return
	}

	// 保存用户信息到会话
	sess.Set(middleware.UsernameKey, username)

	// 设置Cookie
	c.SetCookie(middleware.SessionName, sess.ID, 3600*24, "/", "", false, true)

	logger.Info("用户登录成功", zap.String("username", username))
	c.Redirect(http.StatusFound, "/admin")
}

// Logout 退出登录
func (h *AdminHandler) Logout(c *gin.Context) {
	sessionID, err := c.Cookie(middleware.SessionName)
	if err == nil {
		h.sessionStore.Delete(sessionID)
	}

	c.SetCookie(middleware.SessionName, "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/admin/login")
}

// GetCaptcha 获取验证码
func (h *AdminHandler) GetCaptcha(c *gin.Context) {
	id, b64s, err := captcha.Generate()
	if err != nil {
		logger.Error("生成验证码失败", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	// 将验证码ID保存到Cookie
	c.SetCookie("captcha_id", id, 300, "/", "", false, true)

	// 返回base64字符串
	// 前端会自动添加 data:image/png;base64, 前缀
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, b64s)
}

// ShowDashboard 显示仪表板
func (h *AdminHandler) ShowDashboard(c *gin.Context) {
	ctx := context.Background()

	// 获取统计数据
	accounts, _ := h.crawlerService.GetAccountList(ctx)
	_, total, _ := h.crawlerService.GetArticleList(ctx, "", 1, 10)
	latestArticles, _, _ := h.crawlerService.GetArticleList(ctx, "", 1, 10)

	c.HTML(http.StatusOK, "dashboard", gin.H{
		"Title":    "仪表板",
		"Active":   "dashboard",
		"IsLogin":  true,
		"Username": middleware.GetUsername(c),
		"Stats": gin.H{
			"AccountCount":  len(accounts),
			"ArticleCount":  total,
			"CrawlInterval": viper.GetInt("crawler.interval"),
		},
		"LatestArticles": latestArticles,
	})
}

// ShowAccounts 显示公众号列表
func (h *AdminHandler) ShowAccounts(c *gin.Context) {
	ctx := context.Background()

	accounts, err := h.crawlerService.GetAccountList(ctx)
	if err != nil {
		logger.Error("获取公众号列表失败", zap.Error(err))
	}

	c.HTML(http.StatusOK, "accounts", gin.H{
		"Title":    "公众号管理",
		"Active":   "accounts",
		"IsLogin":  true,
		"Username": middleware.GetUsername(c),
		"Accounts": accounts,
	})
}

// ShowArticles 显示文章列表
func (h *AdminHandler) ShowArticles(c *gin.Context) {
	ctx := context.Background()

	// 获取分页参数
	pageInt, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	page := int64(pageInt)
	pageSize := int64(20)
	accountID := c.Query("account_id")
	keyword := c.Query("keyword")

	// 获取文章列表
	articles, total, err := h.crawlerService.GetArticleList(ctx, accountID, page, pageSize)
	if err != nil {
		logger.Error("获取文章列表失败", zap.Error(err))
	}

	// 如果有搜索关键词，进行过滤
	// TODO: 后期可以在service层实现数据库搜索
	if keyword != "" {
		keyword = strings.ToLower(keyword)
		var filteredArticles []*model.Article
		for _, article := range articles {
			if strings.Contains(strings.ToLower(article.Title), keyword) {
				filteredArticles = append(filteredArticles, article)
			}
		}
		articles = filteredArticles
	}

	// 获取公众号列表（用于筛选）
	accounts, _ := h.crawlerService.GetAccountList(ctx)

	// 计算总页数
	totalPages := int((total + pageSize - 1) / pageSize)

	// 生成页码列表
	var pages []int
	for i := 1; i <= totalPages && i <= 10; i++ {
		pages = append(pages, i)
	}

	c.HTML(http.StatusOK, "articles", gin.H{
		"Title":           "文章管理",
		"Active":          "articles",
		"IsLogin":         true,
		"Username":        middleware.GetUsername(c),
		"Articles":        articles,
		"Accounts":        accounts,
		"Page":            pageInt,
		"TotalPages":      totalPages,
		"Pages":           pages,
		"FilterAccountID": accountID,
		"SearchKeyword":   keyword,
	})
}

// ShowTasks 显示任务管理页面
func (h *AdminHandler) ShowTasks(c *gin.Context) {

	c.HTML(http.StatusOK, "tasks", gin.H{
		"Title":         "任务管理",
		"Active":        "tasks",
		"IsLogin":       true,
		"Username":      middleware.GetUsername(c),
		"CrawlInterval": viper.GetInt("crawler.interval"),
	})
}

// ShowSettings 显示系统设置页面
func (h *AdminHandler) ShowSettings(c *gin.Context) {

	c.HTML(http.StatusOK, "settings", gin.H{
		"Title":         "系统设置",
		"Active":        "settings",
		"IsLogin":       true,
		"Username":      middleware.GetUsername(c),
		"CrawlInterval": viper.GetInt("crawler.interval"),
		"FetchCount":    10, // 默认值
		"Timeout":       viper.GetInt("crawler.timeout"),
		"DatabaseName":  viper.GetString("mongodb.database"),
		"ServerMode":    viper.GetString("server.mode"),
		"ServerPort":    viper.GetString("server.port"),
		"GoVersion":     runtime.Version(),
	})
}

// TriggerCrawl 手动触发爬取任务
func (h *AdminHandler) TriggerCrawl(c *gin.Context) {
	ctx := context.Background()

	logger.Info("手动触发爬取任务", zap.String("operator", middleware.GetUsername(c)))

	go func() {
		if err := h.crawlerService.FetchAllAccounts(ctx); err != nil {
			logger.Error("手动爬取任务失败", zap.Error(err))
		}
	}()

	response.Success(c, gin.H{"msg": "爬取任务已启动"})
}

// UpdateSettings 更新系统设置
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req struct {
		CrawlInterval int `json:"crawl_interval"`
		FetchCount    int `json:"fetch_count"`
		Timeout       int `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误")
		return
	}

	// 验证参数
	if req.CrawlInterval < 5 || req.CrawlInterval > 1440 {
		response.Error(c, http.StatusBadRequest, "爬取间隔必须在5-1440分钟之间")
		return
	}

	if req.FetchCount < 5 || req.FetchCount > 50 {
		response.Error(c, http.StatusBadRequest, "获取文章数量必须在5-50之间")
		return
	}

	if req.Timeout < 30 || req.Timeout > 300 {
		response.Error(c, http.StatusBadRequest, "超时时间必须在30-300秒之间")
		return
	}

	// 更新配置（这里只是演示，实际应该写入配置文件）
	viper.Set("crawler.interval", req.CrawlInterval)
	viper.Set("crawler.timeout", req.Timeout)

	logger.Info("更新系统设置",
		zap.String("operator", middleware.GetUsername(c)),
		zap.Int("interval", req.CrawlInterval),
		zap.Int("timeout", req.Timeout))

	// 写入配置文件
	if err := viper.WriteConfig(); err != nil {
		logger.Error("写入配置文件失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "保存配置失败，但内存配置已更新")
		return
	}

	response.Success(c, gin.H{"msg": "设置已保存，重启服务后生效"})
}

// GetLogs 获取日志内容
func (h *AdminHandler) GetLogs(c *gin.Context) {
	// 获取查询参数
	lines := c.DefaultQuery("lines", "500")  // 读取最后N行
	level := c.DefaultQuery("level", "")     // 日志级别过滤
	keyword := c.DefaultQuery("keyword", "") // 关键词过滤

	linesInt, err := strconv.Atoi(lines)
	if err != nil || linesInt <= 0 {
		linesInt = 500
	}
	if linesInt > 2000 {
		linesInt = 2000 // 限制最大行数
	}

	// 获取项目根目录
	rootDir := getProjectRoot()
	logPath := filepath.Join(rootDir, "cmd", "logs", "app.log")

	// 检查日志文件是否存在
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		response.Error(c, http.StatusNotFound, "日志文件不存在")
		return
	}

	// 读取日志文件
	file, err := os.Open(logPath)
	if err != nil {
		logger.Error("打开日志文件失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "读取日志文件失败")
		return
	}
	defer file.Close()

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		logger.Error("获取日志文件信息失败", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, "读取日志文件失败")
		return
	}
	fileSize := fileInfo.Size()

	// 读取最后N行（使用简单的方法，如果文件很大可以优化）
	var logContent string
	if fileSize > 0 {
		// 估算每行约200字节，计算需要读取的字节数
		estimatedBytes := int64(linesInt * 200)
		if estimatedBytes > fileSize {
			estimatedBytes = fileSize
		}

		// 从文件末尾开始读取
		offset := fileSize - estimatedBytes
		if offset < 0 {
			offset = 0
		}

		_, err = file.Seek(offset, 0)
		if err != nil {
			logger.Error("定位日志文件失败", zap.Error(err))
			response.Error(c, http.StatusInternalServerError, "读取日志文件失败")
			return
		}

		content, err := io.ReadAll(file)
		if err != nil {
			logger.Error("读取日志文件内容失败", zap.Error(err))
			response.Error(c, http.StatusInternalServerError, "读取日志文件失败")
			return
		}

		logContent = string(content)
	}

	// 按行分割
	logLines := strings.Split(logContent, "\n")

	// 只保留最后N行
	if len(logLines) > linesInt {
		logLines = logLines[len(logLines)-linesInt:]
	}

	// 过滤日志级别
	if level != "" {
		var filteredLines []string
		levelUpper := strings.ToUpper(level)
		for _, line := range logLines {
			if strings.Contains(strings.ToUpper(line), levelUpper) {
				filteredLines = append(filteredLines, line)
			}
		}
		logLines = filteredLines
	}

	// 过滤关键词
	if keyword != "" {
		var filteredLines []string
		keywordLower := strings.ToLower(keyword)
		for _, line := range logLines {
			if strings.Contains(strings.ToLower(line), keywordLower) {
				filteredLines = append(filteredLines, line)
			}
		}
		logLines = filteredLines
	}

	response.Success(c, gin.H{
		"logs":      logLines,
		"total":     len(logLines),
		"file_size": fileSize,
		"file_path": logPath,
	})
}

// getProjectRoot 获取项目根目录（包含go.mod的目录）
func getProjectRoot() string {
	// 尝试从当前工作目录开始查找
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// 向上查找直到找到go.mod文件
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// 已经到达根目录
			break
		}
		dir = parent
	}

	return ""
}
