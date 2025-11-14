package middleware

import (
	"net/http"

	"wechat-crawler/pkg/session"

	"github.com/gin-gonic/gin"
)

const (
	SessionName = "wechat_crawler_session"
	UserIDKey   = "user_id"
	UsernameKey = "username"
)

var sessionStore *session.Store

// InitSession 初始化会话存储
func InitSession(store *session.Store) {
	sessionStore = store
}

// AuthRequired 认证中间件
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Cookie中获取session ID
		sessionID, err := c.Cookie(SessionName)
		if err != nil {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// 获取会话
		sess, exists := sessionStore.Get(sessionID)
		if !exists {
			c.SetCookie(SessionName, "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// 检查用户信息
		username := sess.GetString(UsernameKey)
		if username == "" {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// 更新会话过期时间
		sessionStore.Update(sessionID)

		// 将用户信息存入上下文
		c.Set(UsernameKey, username)
		c.Set("session", sess)

		c.Next()
	}
}

// GetSession 从上下文获取会话
func GetSession(c *gin.Context) (*session.Session, bool) {
	sess, exists := c.Get("session")
	if !exists {
		return nil, false
	}
	session, ok := sess.(*session.Session)
	return session, ok
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) string {
	username, exists := c.Get(UsernameKey)
	if !exists {
		return ""
	}
	return username.(string)
}

