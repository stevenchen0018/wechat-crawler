package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FeishuConfig 飞书通知配置
type FeishuConfig struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	WebhookURL   string             `bson:"webhook_url" json:"webhook_url"`     // 飞书webhook地址
	Enabled      bool               `bson:"enabled" json:"enabled"`             // 是否启用通知
	NotifyTime   string             `bson:"notify_time" json:"notify_time"`     // 通知时间，格式：HH:MM，如 "09:00"
	NotifyTitle  string             `bson:"notify_title" json:"notify_title"`   // 通知标题
	NotifyPeriod string             `bson:"notify_period" json:"notify_period"` // 通知周期：daily-每天, hourly-每小时
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// TableName 返回集合名称
func (FeishuConfig) TableName() string {
	return "feishu_config"
}
