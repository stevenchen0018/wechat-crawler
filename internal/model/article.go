package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Article 微信公众号文章
type Article struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AccountID   primitive.ObjectID `bson:"account_id" json:"account_id"`     // 所属公众号ID
	AccountName string             `bson:"account_name" json:"account_name"` // 公众号名称（冗余字段，便于查询）
	Title       string             `bson:"title" json:"title"`               // 文章标题
	Author      string             `bson:"author" json:"author"`             // 作者
	Digest      string             `bson:"digest" json:"digest"`             // 文章摘要
	Content     string             `bson:"content" json:"content"`           // 文章内容（HTML）
	ContentURL  string             `bson:"content_url" json:"content_url"`   // 文章原始URL
	Cover       string             `bson:"cover" json:"cover"`               // 封面图片URL
	SourceURL   string             `bson:"source_url" json:"source_url"`     // 原文链接
	PublishTime int64              `bson:"publish_time" json:"publish_time"` // 发布时间（时间戳）
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`     // 采集时间
}

// TableName 返回集合名称
func (Article) TableName() string {
	return "articles"
}

// ArticleListItem 文章列表项（从微信后台获取的数据结构）
type ArticleListItem struct {
	Aid        string `json:"aid"`         // 文章ID
	Title      string `json:"title"`       // 标题
	Digest     string `json:"digest"`      // 摘要
	Cover      string `json:"cover"`       // 封面
	ContentURL string `json:"link"`        // 文章链接
	CreateTime int64  `json:"create_time"` // 创建时间
	UpdateTime int64  `json:"update_time"` // 更新时间
	Author     string `json:"author"`      // 作者
	SourceURL  string `json:"source_url"`  // 原文链接
}
