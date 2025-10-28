# API 接口文档

## 基本信息

- **Base URL**: `http://localhost:8080`
- **返回格式**: JSON
- **编码**: UTF-8

## 统一响应格式

所有接口返回统一的 JSON 格式：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

### 响应码说明

| Code | 说明 |
|------|------|
| 200  | 成功 |
| 400  | 请求参数错误 |
| 401  | 未授权 |
| 403  | 禁止访问 |
| 404  | 资源不存在 |
| 500  | 服务器内部错误 |

---

## 公众号管理

### 1. 添加公众号订阅

添加一个新的微信公众号订阅

**接口地址**: `POST /api/wechat/add`

**请求参数**:

```json
{
  "name": "公众号名称",
  "alias": "公众号别名（可选）"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 公众号名称 |
| alias | string | 否 | 公众号别名，用于备注 |

**响应示例**:

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "name": "技术公众号",
    "alias": "tech",
    "fake_id": "MzAwMDAwMDAwMA==",
    "url": "",
    "last_article": "",
    "status": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

**错误响应**:

```json
{
  "code": 500,
  "msg": "公众号已存在",
  "data": null
}
```

---

### 2. 获取公众号列表

获取所有已订阅的公众号列表

**接口地址**: `GET /api/wechat/list`

**请求参数**: 无

**响应示例**:

```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": "507f1f77bcf86cd799439011",
      "name": "技术公众号",
      "alias": "tech",
      "fake_id": "MzAwMDAwMDAwMA==",
      "url": "",
      "last_article": "https://mp.weixin.qq.com/s/xxxxx",
      "status": 1,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### 3. 获取公众号详情

根据ID获取公众号详细信息

**接口地址**: `GET /api/wechat/:id`

**路径参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 公众号ID |

**响应示例**:

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "507f1f77bcf86cd799439011",
    "name": "技术公众号",
    "alias": "tech",
    "fake_id": "MzAwMDAwMDAwMA==",
    "url": "",
    "last_article": "https://mp.weixin.qq.com/s/xxxxx",
    "status": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

---

### 4. 删除公众号订阅

取消订阅指定的公众号

**接口地址**: `DELETE /api/wechat/:id`

**路径参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 公众号ID |

**响应示例**:

```json
{
  "code": 200,
  "msg": "删除成功",
  "data": null
}
```

---

## 文章管理

### 5. 获取文章列表

获取文章列表，支持分页和按公众号筛选

**接口地址**: `GET /api/article/list`

**请求参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| account_id | string | 否 | - | 公众号ID，不传则查询所有 |
| page | int | 否 | 1 | 页码 |
| page_size | int | 否 | 20 | 每页数量（1-100） |

**请求示例**:

```
GET /api/article/list?account_id=507f1f77bcf86cd799439011&page=1&page_size=20
```

**响应示例**:

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "list": [
      {
        "id": "507f1f77bcf86cd799439012",
        "account_id": "507f1f77bcf86cd799439011",
        "account_name": "技术公众号",
        "title": "Go语言最佳实践",
        "author": "张三",
        "digest": "本文介绍Go语言的最佳实践...",
        "content": "<html>...</html>",
        "content_url": "https://mp.weixin.qq.com/s/xxxxx",
        "cover": "https://mmbiz.qpic.cn/xxxxx",
        "source_url": "",
        "publish_time": 1704067200,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5
  }
}
```

---

## 爬虫任务

### 6. 手动触发爬取

立即执行一次所有公众号的爬取任务

**接口地址**: `POST /api/crawler/trigger`

**请求参数**: 无

**响应示例**:

```json
{
  "code": 200,
  "msg": "爬取任务已启动",
  "data": null
}
```

**说明**: 该接口会异步执行爬取任务，不会阻塞响应。可通过日志查看执行情况。

---

## 健康检查

### 7. 服务健康检查

检查服务是否正常运行

**接口地址**: `GET /health`

**请求参数**: 无

**响应示例**:

```json
{
  "status": "ok",
  "msg": "wechat-crawler service is running"
}
```

---

## 使用示例

### cURL 示例

**添加公众号**:

```bash
curl -X POST http://localhost:8080/api/wechat/add \
  -H "Content-Type: application/json" \
  -d '{
    "name": "技术公众号",
    "alias": "tech"
  }'
```

**获取文章列表**:

```bash
curl -X GET "http://localhost:8080/api/article/list?page=1&page_size=10"
```

**手动触发爬取**:

```bash
curl -X POST http://localhost:8080/api/crawler/trigger
```

### JavaScript (Fetch) 示例

```javascript
// 添加公众号
fetch('http://localhost:8080/api/wechat/add', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    name: '技术公众号',
    alias: 'tech'
  })
})
.then(response => response.json())
.then(data => console.log(data));

// 获取文章列表
fetch('http://localhost:8080/api/article/list?page=1&page_size=20')
  .then(response => response.json())
  .then(data => console.log(data));
```

### Python 示例

```python
import requests

# 添加公众号
response = requests.post('http://localhost:8080/api/wechat/add', json={
    'name': '技术公众号',
    'alias': 'tech'
})
print(response.json())

# 获取文章列表
response = requests.get('http://localhost:8080/api/article/list', params={
    'page': 1,
    'page_size': 20
})
print(response.json())
```

---

## 注意事项

1. **首次使用**：首次运行系统需要扫码登录微信公众号平台
2. **Cookie有效期**：Cookie通常有效期为几天到几周，过期后需要重新扫码
3. **爬取频率**：建议爬取间隔不低于5分钟，避免频繁请求
4. **并发限制**：默认并发爬取数为3，可在配置文件中调整
5. **数据去重**：系统会自动根据文章URL进行去重

---

## 错误码参考

### 公众号相关错误

| 错误信息 | 原因 | 解决方案 |
|---------|------|---------|
| 公众号已存在 | 该公众号已被订阅 | 检查公众号列表 |
| 未找到公众号 | 搜索不到指定公众号 | 确认公众号名称正确 |
| 无效的ID | ID格式不正确 | 检查ID格式 |

### 爬虫相关错误

| 错误信息 | 原因 | 解决方案 |
|---------|------|---------|
| 获取文章列表失败 | 网络或权限问题 | 检查网络和Cookie有效性 |
| Cookie已失效 | Cookie过期 | 重新扫码登录 |
| 爬取超时 | 网络不稳定 | 增加超时时间或重试 |

---

## 更新日志

### v1.0.0 (2025-10-28)

- 初始版本发布
- 支持公众号订阅管理
- 支持文章自动采集
- 支持定时任务调度
- 提供完整的RESTful API

