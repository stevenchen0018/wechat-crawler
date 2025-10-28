.PHONY: build run clean test help

# 默认目标
.DEFAULT_GOAL := help

# 项目名称
PROJECT_NAME := wechat-crawler
BINARY_NAME := wechat-crawler

# Go 参数
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

## build: 编译项目
build:
	@echo "开始编译..."
	$(GOBUILD) -o $(BINARY_NAME) -v cmd/main.go
	@echo "编译完成！"

## run: 运行项目
run: build
	@echo "启动服务..."
	./$(BINARY_NAME)

## clean: 清理编译文件
clean:
	@echo "清理中..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf ./logs/
	@echo "清理完成！"

## test: 运行测试
test:
	@echo "运行测试..."
	$(GOTEST) -v ./...

## deps: 下载依赖
deps:
	@echo "下载依赖..."
	$(GOMOD) download
	$(GOMOD) tidy

## update: 更新依赖
update:
	@echo "更新依赖..."
	$(GOMOD) tidy
	$(GOGET) -u ./...

## install: 安装到系统
install: build
	@echo "安装到系统..."
	cp $(BINARY_NAME) /usr/local/bin/

## docker-build: 构建Docker镜像
docker-build:
	@echo "构建Docker镜像..."
	docker build -t $(PROJECT_NAME):latest .

## help: 显示帮助信息
help:
	@echo "使用说明："
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

