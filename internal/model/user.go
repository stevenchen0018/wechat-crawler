package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User 管理员用户
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username  string             `bson:"username" json:"username"`
	Password  string             `bson:"password" json:"-"` // 密码不在JSON中返回
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

