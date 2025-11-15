package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"wechat-crawler/internal/model"
	"wechat-crawler/pkg/logger"

	"go.uber.org/zap"
)

// FeishuNotifier 飞书通知器
type FeishuNotifier struct {
	webhookURL string
}

// NewFeishuNotifier 创建飞书通知器
func NewFeishuNotifier(webhookURL string) *FeishuNotifier {
	return &FeishuNotifier{
		webhookURL: webhookURL,
	}
}

// FeishuTextMessage 飞书文本消息
type FeishuTextMessage struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

// FeishuCardMessage 飞书卡片消息
type FeishuCardMessage struct {
	MsgType string      `json:"msg_type"`
	Card    interface{} `json:"card"`
}

// SendTextMessage 发送纯文本消息
func (f *FeishuNotifier) SendTextMessage(text string) error {
	message := FeishuTextMessage{
		MsgType: "text",
	}
	message.Content.Text = text

	return f.sendMessage(message)
}

// SendArticleNotification 发送文章通知
func (f *FeishuNotifier) SendArticleNotification(title string, articles []*model.Article) error {
	if len(articles) == 0 {
		return nil
	}

	// 构建文章列表文本
	var content string
	content += fmt.Sprintf("📢 %s\n\n", title)
	content += fmt.Sprintf("🕐 %s\n", time.Now().Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("📊 共发现 %d 篇新文章\n\n", len(articles))

	// 限制最多显示10篇
	displayCount := len(articles)
	if displayCount > 10 {
		displayCount = 10
	}

	for i := 0; i < displayCount; i++ {
		article := articles[i]
		publishTime := time.Unix(article.PublishTime, 0).Format("2006-01-02 15:04")
		content += fmt.Sprintf("📄 %s\n", article.Title)
		content += fmt.Sprintf("   👤 %s | 📅 %s\n", article.AccountName, publishTime)
		if article.Digest != "" {
			content += fmt.Sprintf("   💬 %s\n", article.Digest)
		}
		content += fmt.Sprintf("   🔗 %s\n\n", article.ContentURL)
	}

	if len(articles) > displayCount {
		content += fmt.Sprintf("... 还有 %d 篇文章未显示", len(articles)-displayCount)
	}

	return f.SendTextMessage(content)
}

// SendArticleCard 发送文章卡片（使用飞书卡片消息）
func (f *FeishuNotifier) SendArticleCard(title string, articles []*model.Article) error {
	if len(articles) == 0 {
		return nil
	}

	// 构建卡片元素
	elements := []interface{}{}

	// 添加标题和统计信息
	elements = append(elements, map[string]interface{}{
		"tag": "div",
		"text": map[string]interface{}{
			"tag":     "lark_md",
			"content": fmt.Sprintf("**%s**\n🕐 %s | 📊 共 %d 篇新文章", title, time.Now().Format("2006-01-02 15:04:05"), len(articles)),
		},
	})

	elements = append(elements, map[string]interface{}{
		"tag": "hr",
	})

	// 限制最多显示5篇
	displayCount := len(articles)
	if displayCount > 5 {
		displayCount = 5
	}

	// 添加文章列表
	for i := 0; i < displayCount; i++ {
		article := articles[i]
		publishTime := time.Unix(article.PublishTime, 0).Format("2006-01-02 15:04")

		contentText := fmt.Sprintf("**[%s](%s)**\n👤 %s | 📅 %s",
			article.Title,
			article.ContentURL,
			article.AccountName,
			publishTime,
		)

		if article.Digest != "" {
			contentText += fmt.Sprintf("\n💬 %s", article.Digest)
		}

		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": contentText,
			},
		})

		if i < displayCount-1 {
			elements = append(elements, map[string]interface{}{
				"tag": "hr",
			})
		}
	}

	if len(articles) > displayCount {
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"tag":     "plain_text",
				"content": fmt.Sprintf("... 还有 %d 篇文章未显示", len(articles)-displayCount),
			},
		})
	}

	// 构建卡片消息
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": title,
			},
		},
		"elements": elements,
	}

	message := FeishuCardMessage{
		MsgType: "interactive",
		Card:    card,
	}

	return f.sendMessage(message)
}

// sendMessage 发送消息到飞书webhook
func (f *FeishuNotifier) sendMessage(message interface{}) error {
	if f.webhookURL == "" {
		return fmt.Errorf("飞书webhook地址未配置")
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	logger.Debug("发送飞书消息", zap.String("webhook", f.webhookURL), zap.String("message", string(jsonData)))

	resp, err := http.Post(f.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("飞书返回错误状态码: %d", resp.StatusCode)
	}

	// 解析飞书响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查返回码
	if code, ok := result["code"].(float64); ok && code != 0 {
		return fmt.Errorf("飞书返回错误: code=%v, msg=%v", result["code"], result["msg"])
	}

	logger.Info("飞书消息发送成功")
	return nil
}

// TestNotification 发送测试通知
func (f *FeishuNotifier) TestNotification() error {
	text := fmt.Sprintf("📢 飞书通知测试\n\n🕐 %s\n✅ 飞书webhook配置正常，通知功能可以正常使用！",
		time.Now().Format("2006-01-02 15:04:05"))
	return f.SendTextMessage(text)
}

