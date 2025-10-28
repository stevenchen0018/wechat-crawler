package crawler

import (
	"encoding/json"
	"fmt"
	"os"

	"wechat-crawler/pkg/logger"

	"go.uber.org/zap"
)

// CookieManager Cookie管理器
type CookieManager struct {
	cookieFile string
}

// NewCookieManager 创建Cookie管理器
func NewCookieManager(cookieFile string) *CookieManager {
	return &CookieManager{
		cookieFile: cookieFile,
	}
}

// SaveCookies 保存Cookies到文件
func (cm *CookieManager) SaveCookies(cookies []*Cookie) error {
	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		logger.Error("序列化Cookie失败", zap.Error(err))
		return err
	}

	err = os.WriteFile(cm.cookieFile, data, 0644)
	if err != nil {
		logger.Error("保存Cookie文件失败", zap.Error(err))
		return err
	}

	logger.Info("Cookie保存成功", zap.String("file", cm.cookieFile))
	return nil
}

// SaveSession 保存会话数据（Cookies + Token）到文件
func (cm *CookieManager) SaveSession(sessionData *SessionData) error {
	data, err := json.MarshalIndent(sessionData, "", "  ")
	if err != nil {
		logger.Error("序列化会话数据失败", zap.Error(err))
		return err
	}

	err = os.WriteFile(cm.cookieFile, data, 0644)
	if err != nil {
		logger.Error("保存会话数据失败", zap.Error(err))
		return err
	}

	logger.Info("会话数据保存成功", zap.String("file", cm.cookieFile), zap.String("token", sessionData.Token))
	return nil
}

// LoadCookies 从文件加载Cookies
func (cm *CookieManager) LoadCookies() ([]*Cookie, error) {
	// 检查文件是否存在
	if _, err := os.Stat(cm.cookieFile); os.IsNotExist(err) {
		logger.Info("Cookie文件不存在，需要首次登录", zap.String("file", cm.cookieFile))
		return nil, nil
	}

	// 读取文件
	data, err := os.ReadFile(cm.cookieFile)
	if err != nil {
		logger.Error("读取Cookie文件失败", zap.Error(err))
		return nil, err
	}

	// 反序列化
	var cookies []*Cookie
	err = json.Unmarshal(data, &cookies)
	if err != nil {
		logger.Error("解析Cookie文件失败", zap.Error(err))
		return nil, err
	}

	logger.Info("Cookie加载成功", zap.Int("count", len(cookies)))
	return cookies, nil
}

// LoadSession 从文件加载会话数据（Cookies + Token）
func (cm *CookieManager) LoadSession() (*SessionData, error) {
	// 检查文件是否存在
	if _, err := os.Stat(cm.cookieFile); os.IsNotExist(err) {
		logger.Info("会话文件不存在，需要首次登录", zap.String("file", cm.cookieFile))
		return nil, nil
	}

	// 读取文件
	data, err := os.ReadFile(cm.cookieFile)
	if err != nil {
		logger.Error("读取会话文件失败", zap.Error(err))
		return nil, err
	}

	// 尝试按SessionData格式解析
	var sessionData SessionData
	if err := json.Unmarshal(data, &sessionData); err == nil && sessionData.Token != "" {
		logger.Info("会话数据加载成功",
			zap.Int("cookie_count", len(sessionData.Cookies)),
			zap.String("token", sessionData.Token))
		return &sessionData, nil
	}

	// 兼容旧格式：如果没有token字段，按旧的Cookie数组格式解析
	var cookies []*Cookie
	if err := json.Unmarshal(data, &cookies); err == nil {
		logger.Info("加载旧格式Cookie（无token）", zap.Int("count", len(cookies)))
		return &SessionData{
			Cookies: cookies,
			Token:   "", // 旧格式没有token
		}, nil
	}

	logger.Error("解析会话文件失败", zap.Error(err))
	return nil, fmt.Errorf("解析会话文件失败")
}

// Cookie Chrome Cookie结构
type Cookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

// SessionData 会话数据（包含Cookies和Token）
type SessionData struct {
	Cookies []*Cookie `json:"cookies"`
	Token   string    `json:"token"`
}

// ToChromeParam 转换为chromedp参数格式（用于设置Cookie）
// 注意：实际使用时通过 network.SetCookie().WithXXX() 方式设置
func (c *Cookie) ToChromeParam() map[string]interface{} {
	return map[string]interface{}{
		"name":     c.Name,
		"value":    c.Value,
		"domain":   c.Domain,
		"path":     c.Path,
		"expires":  c.Expires,
		"httpOnly": c.HTTPOnly,
		"secure":   c.Secure,
	}
}
