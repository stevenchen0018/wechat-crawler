package main

import (
	"fmt"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// 用于生成bcrypt密码哈希的工具程序
func TestGeneratePassword(t *testing.T) {
	password := "admin123"

	// 生成bcrypt哈希
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("生成密码哈希失败: %v\n", err)
		return
	}

	fmt.Println("密码:", password)
	fmt.Println("哈希:", string(hash))
	fmt.Println("\n请将上面的哈希值复制到 config/config.yaml 的 admin.password 字段")
}
