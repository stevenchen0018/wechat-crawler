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

// WeChatAccountRepo 公众号数据访问层
type WeChatAccountRepo struct {
	collection *mongo.Collection
}

// NewWeChatAccountRepo 创建公众号仓库实例
func NewWeChatAccountRepo() *WeChatAccountRepo {
	return &WeChatAccountRepo{
		collection: database.GetCollection(model.WeChatAccount{}.TableName()),
	}
}

// Create 创建公众号记录
func (r *WeChatAccountRepo) Create(ctx context.Context, account *model.WeChatAccount) error {
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()
	account.Status = 1

	result, err := r.collection.InsertOne(ctx, account)
	if err != nil {
		return err
	}

	account.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByID 根据ID查询
func (r *WeChatAccountRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*model.WeChatAccount, error) {
	var account model.WeChatAccount
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&account)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// FindByName 根据名称查询
func (r *WeChatAccountRepo) FindByName(ctx context.Context, name string) (*model.WeChatAccount, error) {
	var account model.WeChatAccount
	err := r.collection.FindOne(ctx, bson.M{"name": name}).Decode(&account)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// FindByFakeID 根据FakeID查询
func (r *WeChatAccountRepo) FindByFakeID(ctx context.Context, fakeID string) (*model.WeChatAccount, error) {
	var account model.WeChatAccount
	err := r.collection.FindOne(ctx, bson.M{"fake_id": fakeID}).Decode(&account)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// List 查询所有公众号
func (r *WeChatAccountRepo) List(ctx context.Context) ([]*model.WeChatAccount, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"status": 1})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var accounts []*model.WeChatAccount
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

// Update 更新公众号信息
func (r *WeChatAccountRepo) Update(ctx context.Context, account *model.WeChatAccount) error {
	account.UpdatedAt = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": account.ID},
		bson.M{"$set": account},
	)
	return err
}

// UpdateLastArticle 更新最后一篇文章URL
func (r *WeChatAccountRepo) UpdateLastArticle(ctx context.Context, id primitive.ObjectID, articleURL string) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"last_article": articleURL,
				"updated_at":   time.Now(),
			},
		},
	)
	return err
}

// Delete 删除公众号（软删除）
func (r *WeChatAccountRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"status":     0,
				"updated_at": time.Now(),
			},
		},
	)
	return err
}

// Count 统计公众号数量
func (r *WeChatAccountRepo) Count(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"status": 1})
}
