package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client   *mongo.Client
	Database *mongo.Database
)

// Config MongoDB配置
type Config struct {
	URI      string
	Database string
	Timeout  int
}

// Init 初始化MongoDB连接
func Init(cfg Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	// 设置客户端选项
	clientOptions := options.Client().ApplyURI(cfg.URI)

	// 连接MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// 验证连接
	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	Client = client
	Database = client.Database(cfg.Database)

	return nil
}

// Close 关闭MongoDB连接
func Close() error {
	if Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return Client.Disconnect(ctx)
	}
	return nil
}

// GetCollection 获取集合
func GetCollection(name string) *mongo.Collection {
	return Database.Collection(name)
}
