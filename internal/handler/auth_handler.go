package handler

import (
	"errors"
	"net/http"
	"regexp"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/pkg/jwt"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type AuthHandler struct {
	cfg  *config.Config
	repo repository.UserRepository
}

func NewAuthHandler(cfg *config.Config, repo repository.UserRepository) *AuthHandler {
	return &AuthHandler{cfg: cfg, repo: repo}
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
		bcrypt.CompareHashAndPassword([]byte("$2a$10$0000000000000000000000000000000000000000000000000000"), []byte(req.Password))
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名只能包含字母、数字和下划线"})
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

	username, _ := c.Get("username")
	user, err := h.repo.GetUserByUsername(username.(string))
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

	if err := h.repo.SetPasswordHash(username.(string), string(hash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码修改失败，请重试"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}
