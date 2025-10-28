package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WeChatAccount 微信公众号账号信息
type WeChatAccount struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`                 // 公众号名称
	Alias       string             `bson:"alias" json:"alias"`               // 公众号别名（用于搜索）
	FakeID      string             `bson:"fake_id" json:"fake_id"`           // 微信公众号的唯一标识
	URL         string             `bson:"url" json:"url"`                   // 公众号主页URL
	LastArticle string             `bson:"last_article" json:"last_article"` // 最后一篇文章的URL（用于判断是否有新文章）
	Status      int                `bson:"status" json:"status"`             // 状态：1-正常 0-禁用
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`     // 创建时间
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`     // 更新时间
}

// TableName 返回集合名称
func (WeChatAccount) TableName() string {
	return "wechat_accounts"
}
