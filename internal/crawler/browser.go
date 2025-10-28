package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"wechat-crawler/internal/model"
	"wechat-crawler/pkg/logger"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"
)

// Browser 浏览器封装
type Browser struct {
	ctx           context.Context
	cancel        context.CancelFunc
	cookieManager *CookieManager
	mpURL         string
	timeout       time.Duration
	token         string // 微信公众号平台的token，用于API请求
	debugMode     bool   // debug模式，为true时浏览器不自动关闭
}

// NewBrowser 创建浏览器实例
func NewBrowser(cookieFile, mpURL string, timeout int, debugMode bool) (*Browser, error) {
	// 创建chromedp上下文
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // 首次登录需要显示浏览器窗口
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-dev-shm-usage", true), // 提高稳定性
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	var allocCtx context.Context
	var cancel context.CancelFunc

	if debugMode {
		// Debug模式：使用独立的context，不受程序生命周期影响
		logger.Info("🔍 Debug模式已启用：浏览器将保持打开直到手动关闭")
		allocCtx, cancel = chromedp.NewExecAllocator(context.Background(), opts...)
	} else {
		// 正常模式
		allocCtx, cancel = chromedp.NewExecAllocator(context.Background(), opts...)
	}

	ctx, _ := chromedp.NewContext(allocCtx, chromedp.WithLogf(logger.Logger.Sugar().Debugf))

	browser := &Browser{
		ctx:           ctx,
		cancel:        cancel,
		cookieManager: NewCookieManager(cookieFile),
		mpURL:         mpURL,
		timeout:       time.Duration(timeout) * time.Second,
		debugMode:     debugMode,
	}

	return browser, nil
}

// Close 关闭浏览器
func (b *Browser) Close() {
	if b.debugMode {
		logger.Info("⚠️  Debug模式开启，浏览器不会自动关闭，请手动关闭")
		return
	}

	if b.cancel != nil {
		b.cancel()
	}
}

// Login 登录微信公众号平台
func (b *Browser) Login() error {
	logger.Info("开始登录微信公众号平台")

	// 尝试加载已有的会话数据（Cookies + Token）
	sessionData, err := b.cookieManager.LoadSession()
	if err == nil && sessionData != nil && len(sessionData.Cookies) > 0 {
		logger.Info("尝试使用已保存的会话数据登录")
		if err := b.loginWithCookies(sessionData.Cookies); err == nil {
			// Cookie有效，保存token
			b.token = sessionData.Token
			logger.Info("使用已保存的token", zap.String("token", b.token))
			return nil
		}
		logger.Warn("Cookie登录失败，需要重新扫码登录")
	}

	// Cookie不存在或失效，需要扫码登录
	return b.loginWithQRCode()
}

// loginWithCookies 使用Cookie登录
func (b *Browser) loginWithCookies(cookies []*Cookie) error {
	// Debug模式下不使用超时context，避免函数返回后浏览器被关闭
	var ctx context.Context
	var cancel context.CancelFunc

	if b.debugMode {
		// Debug模式：使用浏览器的主context，不会自动取消
		ctx = b.ctx
		cancel = func() {} // 空函数，不执行任何操作
	} else {
		// 正常模式：使用带超时的context
		ctx, cancel = context.WithTimeout(b.ctx, b.timeout)
	}
	defer cancel()

	// 首先访问页面以建立域
	err := chromedp.Run(ctx, chromedp.Navigate(b.mpURL))
	if err != nil {
		return err
	}

	// 设置所有Cookie
	for _, cookie := range cookies {
		// 捕获当前cookie的值用于闭包
		c := cookie
		err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			// 将 float64 时间戳转换为 time.Time，然后转为 TimeSinceEpoch
			// cookie.Expires 是 Unix 秒级时间戳
			expireTime := cdp.TimeSinceEpoch(time.Unix(int64(c.Expires), 0))
			return network.SetCookie(c.Name, c.Value).
				WithDomain(c.Domain).
				WithPath(c.Path).
				WithExpires(&expireTime).
				WithHTTPOnly(c.HTTPOnly).
				WithSecure(c.Secure).
				Do(ctx)
		}))
		if err != nil {
			logger.Warn("设置Cookie失败", zap.String("name", cookie.Name), zap.Error(err))
		}
	}

	// 重新加载页面使Cookie生效
	err = chromedp.Run(ctx,
		chromedp.Navigate(b.mpURL),
		chromedp.Sleep(2*time.Second),
	)

	if err != nil {
		return err
	}

	// 检查是否登录成功（通过判断URL是否跳转）
	var currentURL string
	err = chromedp.Run(ctx, chromedp.Location(&currentURL))
	if err != nil {
		return err
	}

	if strings.Contains(currentURL, "home") || strings.Contains(currentURL, "cgi-bin") {
		logger.Info("Cookie登录成功", zap.String("url", currentURL))

		// 如果当前token为空，尝试从URL中提取token
		if b.token == "" {
			u, err := url.Parse(currentURL)
			if err == nil {
				query := u.Query()
				token := query.Get("token")
				if token != "" {
					b.token = token
					logger.Info("从URL中提取token", zap.String("token", b.token))

					// 更新保存的会话数据
					sessionData, _ := b.cookieManager.LoadSession()
					if sessionData != nil {
						sessionData.Token = b.token
						b.cookieManager.SaveSession(sessionData)
					}
				}
			}
		}

		return nil
	}

	return fmt.Errorf("cookie已失效")
}

// loginWithQRCode 扫码登录
func (b *Browser) loginWithQRCode() error {
	logger.Info("请使用微信扫码登录公众号平台")

	// Debug模式下不使用超时，允许用户慢慢扫码和调试
	var ctx context.Context
	var cancel context.CancelFunc

	if b.debugMode {
		ctx = b.ctx
		cancel = func() {}
		logger.Info("🔍 Debug模式：扫码登录不限时，可以随时调试")
	} else {
		ctx, cancel = context.WithTimeout(b.ctx, 120*time.Second)
	}
	defer cancel()

	var cookies []*network.Cookie

	// 访问登录页面
	err := chromedp.Run(ctx,
		chromedp.Navigate(b.mpURL),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		logger.Error("访问登录页面失败", zap.Error(err))
		return err
	}

	// 等待登录成功（轮询检查URL变化）
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var timeoutChan <-chan time.Time
	if b.debugMode {
		// Debug模式：不设置超时
		timeoutChan = make(<-chan time.Time) // 永不触发的channel
	} else {
		// 正常模式：120秒超时
		timeoutChan = time.After(120 * time.Second)
	}

	var currentURL string
	for {
		select {
		case <-timeoutChan:
			return fmt.Errorf("扫码登录超时")
		case <-ticker.C:
			err := chromedp.Run(ctx, chromedp.Location(&currentURL))
			if err == nil && (strings.Contains(currentURL, "home") || strings.Contains(currentURL, "cgi-bin")) {
				logger.Info("检测到登录成功", zap.String("url", currentURL))
				goto LoginSuccess
			}
		}
	}

LoginSuccess:
	// 等待页面稳定
	time.Sleep(1 * time.Second)

	// 获取登录后的Cookie
	err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		cookies, err = network.GetCookies().Do(ctx)
		return err
	}))

	if err != nil {
		logger.Error("扫码登录失败", zap.Error(err))
		return err
	}
	// 从URL中提取token
	u, err := url.Parse(currentURL)
	if err != nil {
		logger.Error("解析URL失败", zap.Error(err))
		return err
	}

	query := u.Query()
	token := query.Get("token")
	if token == "" {
		logger.Warn("未能从URL中获取token", zap.String("url", currentURL))
	} else {
		logger.Info("成功获取token", zap.String("token", token))
		b.token = token // 保存到Browser实例
	}

	// 转换并保存Cookie和Token
	simpleCookies := make([]*Cookie, 0, len(cookies))
	for _, c := range cookies {
		simpleCookies = append(simpleCookies, &Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  float64(c.Expires),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
			SameSite: c.SameSite.String(),
		})
	}

	// 保存会话数据（Cookies + Token）
	sessionData := &SessionData{
		Cookies: simpleCookies,
		Token:   b.token,
	}

	if err := b.cookieManager.SaveSession(sessionData); err != nil {
		logger.Warn("保存会话数据失败", zap.Error(err))
	}

	logger.Info("扫码登录成功", zap.String("token", b.token))
	return nil
}

// SearchAccount 搜索公众号并获取FakeID
func (b *Browser) SearchAccount(accountName string) (string, error) {
	logger.Info("搜索公众号", zap.String("name", accountName), zap.String("token", b.token))

	// 检查token是否存在
	if b.token == "" {
		return "", fmt.Errorf("token为空，请先登录")
	}

	// Debug模式下不使用超时，方便调试
	var ctx context.Context
	var cancel context.CancelFunc

	if b.debugMode {
		ctx = b.ctx
		cancel = func() {}
		logger.Debug("Debug模式：SearchAccount 不使用超时限制")
	} else {
		ctx, cancel = context.WithTimeout(b.ctx, b.timeout)
	}
	defer cancel()

	// 构造搜索URL，使用保存的token
	searchURL := fmt.Sprintf("%s/cgi-bin/searchbiz?action=search_biz&begin=0&count=5&query=%s&token=%s&lang=zh_CN&f=json&ajax=1",
		b.mpURL, url.QueryEscape(accountName), b.token)

	logger.Info("搜索URL", zap.String("url", searchURL))

	var responseText string
	err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.Sleep(2*time.Second),
		chromedp.Text("body", &responseText, chromedp.ByQuery),
	)

	if err != nil {
		logger.Error("搜索公众号失败", zap.Error(err))
		return "", err
	}

	logger.Info("搜索公众号响应", zap.String("responseText", responseText))

	// 解析响应（这里需要根据实际返回的JSON结构来解析）
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		logger.Error("解析搜索结果失败", zap.Error(err))
		return "", err
	}

	// 检查是否有错误信息
	if baseResp, ok := result["base_resp"].(map[string]interface{}); ok {
		if ret, ok := baseResp["ret"].(float64); ok && ret != 0 {
			errMsg := baseResp["err_msg"].(string)
			logger.Error("搜索失败", zap.Float64("ret", ret), zap.String("err_msg", errMsg))
			return "", fmt.Errorf("搜索失败: %s (ret=%v)", errMsg, ret)
		}
	}

	// 提取fakeID（实际需要根据微信返回的数据结构调整）
	if list, ok := result["list"].([]interface{}); ok && len(list) > 0 {
		if item, ok := list[0].(map[string]interface{}); ok {
			if fakeID, ok := item["fakeid"].(string); ok {
				logger.Info("找到公众号", zap.String("fakeID", fakeID))
				return fakeID, nil
			}
		}
	}

	return "", fmt.Errorf("未找到公众号: %s", accountName)
}

// FetchArticles 获取公众号文章列表
func (b *Browser) FetchArticles(fakeID string, count int) ([]*model.ArticleListItem, error) {
	logger.Info("获取文章列表", zap.String("fakeID", fakeID), zap.Int("count", count), zap.String("token", b.token))

	// 检查token是否存在
	if b.token == "" {
		return nil, fmt.Errorf("token为空，请先登录")
	}

	// Debug模式下不使用超时，方便调试
	var ctx context.Context
	var cancel context.CancelFunc

	if b.debugMode {
		ctx = b.ctx
		cancel = func() {}
		logger.Debug("Debug模式：FetchArticles 不使用超时限制")
	} else {
		ctx, cancel = context.WithTimeout(b.ctx, b.timeout)
	}
	defer cancel()

	// 构造文章列表URL，使用保存的token
	articleURL := fmt.Sprintf("%s/cgi-bin/appmsg?action=list_ex&begin=0&count=%d&fakeid=%s&type=9&token=%s",
		b.mpURL, count, fakeID, b.token)

	logger.Info("文章列表URL", zap.String("url", articleURL))

	var responseText string
	err := chromedp.Run(ctx,
		chromedp.Navigate(articleURL),
		chromedp.Sleep(2*time.Second),
		chromedp.Text("body", &responseText, chromedp.ByQuery),
	)

	if err != nil {
		logger.Error("获取文章列表失败", zap.Error(err))
		return nil, err
	}

	logger.Info("文章列表响应", zap.String("responseText", responseText))

	// 解析响应
	var result struct {
		BaseResp   map[string]interface{}   `json:"base_resp"`
		AppMsgList []*model.ArticleListItem `json:"app_msg_list"`
	}

	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		logger.Error("解析文章列表失败", zap.Error(err))
		return nil, err
	}

	// 检查是否有错误信息
	if result.BaseResp != nil {
		if ret, ok := result.BaseResp["ret"].(float64); ok && ret != 0 {
			errMsg, _ := result.BaseResp["err_msg"].(string)
			logger.Error("获取文章列表失败", zap.Float64("ret", ret), zap.String("err_msg", errMsg))
			return nil, fmt.Errorf("获取文章列表失败: %s (ret=%v)", errMsg, ret)
		}
	}

	logger.Info("获取文章列表成功", zap.Int("count", len(result.AppMsgList)))
	return result.AppMsgList, nil
}

// FetchArticleContent 获取文章详细内容
func (b *Browser) FetchArticleContent(articleURL string) (string, error) {
	logger.Info("获取文章内容", zap.String("url", articleURL))

	// Debug模式下不使用超时，方便调试
	var ctx context.Context
	var cancel context.CancelFunc

	if b.debugMode {
		ctx = b.ctx
		cancel = func() {}
	} else {
		ctx, cancel = context.WithTimeout(b.ctx, b.timeout)
	}
	defer cancel()

	var content string
	err := chromedp.Run(ctx,
		chromedp.Navigate(articleURL),
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML("#js_content", &content, chromedp.ByID),
	)

	if err != nil {
		logger.Error("获取文章内容失败", zap.Error(err))
		return "", err
	}

	return content, nil
}

// GetToken 从当前页面提取token（用于API请求）
func (b *Browser) GetToken() (string, error) {
	// Debug模式下不使用超时
	var ctx context.Context
	var cancel context.CancelFunc

	if b.debugMode {
		ctx = b.ctx
		cancel = func() {}
	} else {
		ctx, cancel = context.WithTimeout(b.ctx, 10*time.Second)
	}
	defer cancel()

	var token string
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`window.wx.data.t || ""`, &token),
	)

	if err != nil || token == "" {
		logger.Warn("无法获取token", zap.Error(err))
		return "", fmt.Errorf("获取token失败")
	}

	return token, nil
}
