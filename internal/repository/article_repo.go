package repository

import (
	"context"
	"time"

	"wechat-crawler/internal/model"
	"wechat-crawler/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ArticleRepo 文章数据访问层
type ArticleRepo struct {
	collection *mongo.Collection
}

// NewArticleRepo 创建文章仓库实例
func NewArticleRepo() *ArticleRepo {
	return &ArticleRepo{
		collection: database.GetCollection(model.Article{}.TableName()),
	}
}

// Create 创建文章记录
func (r *ArticleRepo) Create(ctx context.Context, article *model.Article) error {
	article.CreatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, article)
	if err != nil {
		return err
	}

	article.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// BatchCreate 批量创建文章
func (r *ArticleRepo) BatchCreate(ctx context.Context, articles []*model.Article) error {
	if len(articles) == 0 {
		return nil
	}

	docs := make([]interface{}, len(articles))
	for i, article := range articles {
		article.CreatedAt = time.Now()
		docs[i] = article
	}

	_, err := r.collection.InsertMany(ctx, docs)
	return err
}

// FindByID 根据ID查询
func (r *ArticleRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Article, error) {
	var article model.Article
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&article)
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// FindByContentURL 根据文章URL查询（用于去重）
func (r *ArticleRepo) FindByContentURL(ctx context.Context, contentURL string) (*model.Article, error) {
	var article model.Article
	err := r.collection.FindOne(ctx, bson.M{"content_url": contentURL}).Decode(&article)
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// ExistsByContentURL 检查文章是否已存在
func (r *ArticleRepo) ExistsByContentURL(ctx context.Context, contentURL string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"content_url": contentURL})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListByAccountID 根据公众号ID查询文章列表
func (r *ArticleRepo) ListByAccountID(ctx context.Context, accountID primitive.ObjectID, page, pageSize int64) ([]*model.Article, int64, error) {
	// 计算总数
	total, err := r.collection.CountDocuments(ctx, bson.M{"account_id": accountID})
	if err != nil {
		return nil, 0, err
	}

	// 查询列表
	opts := options.Find().
		SetSort(bson.D{{Key: "publish_time", Value: -1}}).
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize)

	cursor, err := r.collection.Find(ctx, bson.M{"account_id": accountID}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var articles []*model.Article
	if err := cursor.All(ctx, &articles); err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// ListWithFilter 根据条件查询文章列表（支持关键词、时间范围、公众号筛选）
func (r *ArticleRepo) ListWithFilter(ctx context.Context, accountID string, keyword string, startTime, endTime int64, page, pageSize int64) ([]*model.Article, int64, error) {
	// 构建查询条件
	filter := bson.M{}

	// 公众号筛选
	if accountID != "" {
		objectID, err := primitive.ObjectIDFromHex(accountID)
		if err == nil {
			filter["account_id"] = objectID
		}
	}

	// 关键词搜索（标题）
	if keyword != "" {
		filter["title"] = bson.M{"$regex": keyword, "$options": "i"} // 不区分大小写
	}

	// 时间范围筛选
	if startTime > 0 || endTime > 0 {
		timeFilter := bson.M{}
		if startTime > 0 {
			timeFilter["$gte"] = startTime
		}
		if endTime > 0 {
			timeFilter["$lte"] = endTime
		}
		filter["publish_time"] = timeFilter
	}

	// 计算总数
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// 查询列表
	opts := options.Find().
		SetSort(bson.D{{Key: "publish_time", Value: -1}}).
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var articles []*model.Article
	if err := cursor.All(ctx, &articles); err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// List 查询所有文章（分页）
func (r *ArticleRepo) List(ctx context.Context, page, pageSize int64) ([]*model.Article, int64, error) {
	// 计算总数
	total, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	// 查询列表
	opts := options.Find().
		SetSort(bson.D{{Key: "publish_time", Value: -1}}).
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize)

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var articles []*model.Article
	if err := cursor.All(ctx, &articles); err != nil {
		return nil, 0, err
	}

	return articles, total, nil
}

// GetLatestByAccountID 获取公众号最新的一篇文章
func (r *ArticleRepo) GetLatestByAccountID(ctx context.Context, accountID primitive.ObjectID) (*model.Article, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "publish_time", Value: -1}})

	var article model.Article
	err := r.collection.FindOne(ctx, bson.M{"account_id": accountID}, opts).Decode(&article)
	if err != nil {
		return nil, err
	}
	return &article, nil
}

// CountByAccountID 统计公众号文章数量
func (r *ArticleRepo) CountByAccountID(ctx context.Context, accountID primitive.ObjectID) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"account_id": accountID})
}

// Delete 删除文章
func (r *ArticleRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
