package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code int         `json:"code"`           // 状态码
	Msg  string      `json:"msg"`            // 消息
	Data interface{} `json:"data,omitempty"` // 数据
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: 200,
		Msg:  "success",
		Data: data,
	})
}

// SuccessWithMsg 成功响应（自定义消息）
func SuccessWithMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: 200,
		Msg:  msg,
		Data: data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}

// ErrorWithData 错误响应（带数据）
func ErrorWithData(c *gin.Context, code int, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

// BadRequest 请求参数错误
func BadRequest(c *gin.Context, msg string) {
	Error(c, 400, msg)
}

// Unauthorized 未授权
func Unauthorized(c *gin.Context, msg string) {
	Error(c, 401, msg)
}

// Forbidden 禁止访问
func Forbidden(c *gin.Context, msg string) {
	Error(c, 403, msg)
}

// NotFound 资源不存在
func NotFound(c *gin.Context, msg string) {
	Error(c, 404, msg)
}

// InternalServerError 服务器内部错误
func InternalServerError(c *gin.Context, msg string) {
	Error(c, 500, msg)
}

// PageData 分页数据结构
type PageData struct {
	List       interface{} `json:"list"`        // 数据列表
	Total      int64       `json:"total"`       // 总数
	Page       int64       `json:"page"`        // 当前页
	PageSize   int64       `json:"page_size"`   // 每页数量
	TotalPages int64       `json:"total_pages"` // 总页数
}

// SuccessWithPage 分页成功响应
func SuccessWithPage(c *gin.Context, list interface{}, total, page, pageSize int64) {
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	Success(c, PageData{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}
