package repository

import (
	"context"
	"time"

	"wechat-crawler/internal/model"
	"wechat-crawler/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// FeishuConfigRepo 飞书配置数据访问层
type FeishuConfigRepo struct {
	collection *mongo.Collection
}

// NewFeishuConfigRepo 创建飞书配置仓库实例
func NewFeishuConfigRepo() *FeishuConfigRepo {
	return &FeishuConfigRepo{
		collection: database.GetCollection(model.FeishuConfig{}.TableName()),
	}
}

// GetConfig 获取飞书配置（只有一条配置）
func (r *FeishuConfigRepo) GetConfig(ctx context.Context) (*model.FeishuConfig, error) {
	var config model.FeishuConfig
	err := r.collection.FindOne(ctx, bson.M{}).Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// 返回默认配置
			return &model.FeishuConfig{
				Enabled:      false,
				NotifyTime:   "09:00",
				NotifyTitle:  "微信公众号文章推送",
				NotifyPeriod: "daily",
			}, nil
		}
		return nil, err
	}
	return &config, nil
}

// SaveConfig 保存飞书配置
func (r *FeishuConfigRepo) SaveConfig(ctx context.Context, config *model.FeishuConfig) error {
	config.UpdatedAt = time.Now()

	// 如果ID为空，表示新建
	if config.ID.IsZero() {
		config.CreatedAt = time.Now()
		result, err := r.collection.InsertOne(ctx, config)
		if err != nil {
			return err
		}
		config.ID = result.InsertedID.(primitive.ObjectID)
		return nil
	}

	// 更新现有配置
	filter := bson.M{"_id": config.ID}
	update := bson.M{
		"$set": bson.M{
			"webhook_url":   config.WebhookURL,
			"enabled":       config.Enabled,
			"notify_time":   config.NotifyTime,
			"notify_title":  config.NotifyTitle,
			"notify_period": config.NotifyPeriod,
			"updated_at":    config.UpdatedAt,
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// UpsertConfig 更新或插入配置（确保只有一条配置记录）
func (r *FeishuConfigRepo) UpsertConfig(ctx context.Context, config *model.FeishuConfig) error {
	config.UpdatedAt = time.Now()

	// 查找现有配置
	existingConfig, err := r.GetConfig(ctx)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}

	// 如果存在配置，使用其ID
	if existingConfig != nil && !existingConfig.ID.IsZero() {
		config.ID = existingConfig.ID
	}

	// 如果配置不存在，创建新配置
	if config.ID.IsZero() {
		config.CreatedAt = time.Now()
		result, err := r.collection.InsertOne(ctx, config)
		if err != nil {
			return err
		}
		config.ID = result.InsertedID.(primitive.ObjectID)
		return nil
	}

	// 更新现有配置
	filter := bson.M{"_id": config.ID}
	update := bson.M{
		"$set": bson.M{
			"webhook_url":   config.WebhookURL,
			"enabled":       config.Enabled,
			"notify_time":   config.NotifyTime,
			"notify_title":  config.NotifyTitle,
			"notify_period": config.NotifyPeriod,
			"updated_at":    config.UpdatedAt,
		},
	}

	_, err = r.collection.UpdateOne(ctx, filter, update)
	return err
}
