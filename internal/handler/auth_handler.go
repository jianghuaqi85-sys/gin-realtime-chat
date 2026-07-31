package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/pkg/jwt"
	"github.com/example/gin-high-performance/pkg/ws"
)

var dummyBcryptHash []byte

func init() {
	dummyBcryptHash, _ = bcrypt.GenerateFromPassword([]byte("dummy_password"), bcrypt.DefaultCost)
}

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\p{Han}]+$`)

type AuthHandler struct {
	cfg  *config.Config
	repo repository.UserRepository
	hub  *ws.Hub
}

func NewAuthHandler(cfg *config.Config, repo repository.UserRepository, hub *ws.Hub) *AuthHandler {
	return &AuthHandler{cfg: cfg, repo: repo, hub: hub}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type RegisterRequest struct {
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.repo.GetUserByUsername(req.Username)
	if err != nil || user == nil {
		// 防御计时攻击：使用全局合法的 dummyBcryptHash 模拟完满的计算耗时
		_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(req.Password))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	if user.Banned {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已被封禁"})
		return
	}

	token, err := jwt.GenerateToken(
		h.cfg.JWTSecret,
		user.ID,
		user.Username,
		user.Role,
		h.cfg.JWTExpireHours,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败，请重试"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "两次输入的密码不一致"})
		return
	}

	if nameLen := utf8.RuneCountInString(req.Username); nameLen < 3 || nameLen > 32 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度需要 3-32 个字符"})
		return
	}
	if !usernameRegex.MatchString(req.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名只能包含字母、数字、下划线和中文"})
		return
	}
	if pwdLen := utf8.RuneCountInString(req.Password); pwdLen < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码长度不能少于 8 个字符"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，请重试"})
		return
	}

	if err := h.repo.CreateUser(req.Username, string(hash)); err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "用户名已被占用"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，请重试"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pwdLen := utf8.RuneCountInString(req.NewPassword); pwdLen < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码长度不能少于 8 个字符"})
		return
	}

	usernameVal, _ := c.Get("username")
	username, ok := usernameVal.(string)
	if !ok || username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	user, err := h.repo.GetUserByUsername(username)
	if err != nil || user == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "旧密码不正确"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败，请重试"})
		return
	}

	if err := h.repo.SetPasswordHash(username, string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码修改失败，请重试"})
		return
	}

	// 密码修改成功，切断现有 WS 链接提示重新登录
	if h.hub != nil {
		h.hub.DisconnectUser(user.ID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

type ChangeUsernameRequest struct {
	NewUsername string `json:"new_username" binding:"required"`
}

func (h *AuthHandler) ChangeUsername(c *gin.Context) {
	var req ChangeUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newUsername := req.NewUsername
	if nameLen := utf8.RuneCountInString(newUsername); nameLen < 3 || nameLen > 32 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度需要 3-32 个字符"})
		return
	}
	if !usernameRegex.MatchString(newUsername) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名只能包含字母、数字、下划线和中文"})
		return
	}

	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	if err := h.repo.SetUsername(userID, newUsername); err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "用户名已被占用"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户名修改失败"})
		return
	}

	// 更新 WebSocket Hub 中的用户名
	if h.hub != nil {
		h.hub.UpdateUsername(userID, newUsername)

		if updateMsg, err := json.Marshal(ws.WSMessage{
			Type:    "username_updated",
			UserID:  userID,
			Content: newUsername,
		}); err == nil {
			h.hub.Broadcast(updateMsg)
		}
	}

	// 优化：优先从 gin.Context 中获取角色，减少查库 IO
	roleStr := "user"
	if roleVal, exists := c.Get("role"); exists {
		if r, ok := roleVal.(string); ok && r != "" {
			roleStr = r
		}
	}

	newToken, err := jwt.GenerateToken(
		h.cfg.JWTSecret,
		userID,
		newUsername,
		roleStr,
		h.cfg.JWTExpireHours,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 token 失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户名修改成功", "token": newToken})
}
