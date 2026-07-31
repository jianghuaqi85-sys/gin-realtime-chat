package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/pkg/ws"
)

type AdminHandler struct {
	userRepo    repository.UserRepository
	channelRepo repository.ChannelRepository
	messageRepo repository.MessageRepository
	hub         *ws.Hub
}

func NewAdminHandler(userRepo repository.UserRepository, channelRepo repository.ChannelRepository, messageRepo repository.MessageRepository, hub *ws.Hub) *AdminHandler {
	return &AdminHandler{
		userRepo:    userRepo,
		channelRepo: channelRepo,
		messageRepo: messageRepo,
		hub:         hub,
	}
}

// GET /api/admin/stats
func (h *AdminHandler) Stats(c *gin.Context) {
	ctx := c.Request.Context()
	eg, _ := errgroup.WithContext(ctx)

	var userCount, chCount, msgCount int64

	if h.userRepo != nil {
		eg.Go(func() error {
			var err error
			userCount, err = h.userRepo.Count()
			return err
		})
	}
	if h.channelRepo != nil {
		eg.Go(func() error {
			var err error
			chCount, err = h.channelRepo.Count()
			return err
		})
	}
	if h.messageRepo != nil {
		eg.Go(func() error {
			var err error
			msgCount, err = h.messageRepo.Count()
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计数据失败"})
		return
	}

	online := 0
	if h.hub != nil {
		online = h.hub.OnlineCount()
	}

	c.JSON(http.StatusOK, gin.H{
		"users_total": userCount,
		"online":      online,
		"channels":    chCount,
		"messages":    msgCount,
	})
}

// GET /api/admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	if h.userRepo == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	users, err := h.userRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}
	type userView struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
		Banned   bool   `json:"banned"`
	}
	result := make([]userView, len(users))
	for i, u := range users {
		result[i] = userView{ID: u.ID, Username: u.Username, Role: u.Role, Banned: u.Banned}
	}
	c.JSON(http.StatusOK, result)
}

// DELETE /api/admin/users/:id
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	// 顺序优化：先删除 DB 数据（原子操作），成功后再处理断连副作用
	if h.userRepo != nil {
		if err := h.userRepo.Delete(userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败"})
			return
		}
	}

	if h.hub != nil {
		h.hub.DisconnectUser(userID)
	}

	c.JSON(http.StatusOK, gin.H{"deleted": userID})
}

// POST /api/admin/set-admin {user_id}
func (h *AdminHandler) SetAdmin(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.userRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "repository not initialized"})
		return
	}

	user, err := h.userRepo.GetUserByID(req.UserID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	currentUserID, _ := c.Get("user_id")
	if currentUserIDStr, ok := currentUserID.(string); ok && currentUserIDStr == req.UserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能给自己设置管理员"})
		return
	}

	if err := h.userRepo.SetRole(req.UserID, "admin"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置管理员失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "设置管理员成功", "user_id": req.UserID})
}

// POST /api/admin/remove-admin {user_id}
func (h *AdminHandler) RemoveAdmin(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.userRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "repository not initialized"})
		return
	}

	user, err := h.userRepo.GetUserByID(req.UserID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	currentUserID, _ := c.Get("user_id")
	if currentUserIDStr, ok := currentUserID.(string); ok && currentUserIDStr == req.UserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能撤销自己的管理员权限"})
		return
	}

	if user.Role == "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "不能撤销超级管理员的权限"})
		return
	}

	if err := h.userRepo.SetRole(req.UserID, "user"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "撤销管理员权限失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "撤销管理员权限成功", "user_id": req.UserID})
}

// POST /api/admin/ban {user_id}
func (h *AdminHandler) Ban(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.userRepo != nil {
		if err := h.userRepo.SetBanned(req.UserID, true); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "封禁用户失败"})
			return
		}
	}

	if h.hub != nil {
		if banMsg, err := json.Marshal(ws.WSMessage{
			Type:    "user_banned",
			UserID:  req.UserID,
			Content: req.UserID,
		}); err == nil {
			h.hub.Broadcast(banMsg)
		}
		h.hub.DisconnectUser(req.UserID)
	}

	c.JSON(http.StatusOK, gin.H{"banned": req.UserID})
}

// POST /api/admin/unban {user_id}
func (h *AdminHandler) Unban(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.userRepo != nil {
		if err := h.userRepo.SetBanned(req.UserID, false); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解封用户失败"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"unbanned": req.UserID})
}

// DELETE /api/admin/channels/:id
func (h *AdminHandler) DeleteChannel(c *gin.Context) {
	id := c.Param("id")
	if h.channelRepo != nil {
		if err := h.channelRepo.Delete(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除频道失败"})
			return
		}
	}

	if h.hub != nil {
		if deleteMsg, err := json.Marshal(ws.WSMessage{
			Type:      "channel_deleted",
			ChannelID: id,
		}); err == nil {
			h.hub.Broadcast(deleteMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// DELETE /api/admin/channels/:id/messages
func (h *AdminHandler) ClearMessages(c *gin.Context) {
	channelID := c.Param("id")
	if h.messageRepo != nil {
		if err := h.messageRepo.DeleteByChannel(channelID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "清空消息失败"})
			return
		}
	}

	if h.hub != nil {
		if clearMsg, err := json.Marshal(ws.WSMessage{
			Type:      "messages_cleared",
			ChannelID: channelID,
		}); err == nil {
			h.hub.BroadcastToChannel(channelID, clearMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{"cleared": channelID})
}

// DELETE /api/admin/messages/:id
func (h *AdminHandler) DeleteMessage(c *gin.Context) {
	id := c.Param("id")

	if h.messageRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "repository not initialized"})
		return
	}

	msg, err := h.messageRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
		return
	}

	if err := h.messageRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除消息失败"})
		return
	}

	if h.hub != nil {
		if deleteMsg, err := json.Marshal(ws.WSMessage{
			Type:      "message_deleted",
			ChannelID: msg.ChannelID,
			Content:   id,
		}); err == nil {
			h.hub.BroadcastToChannel(msg.ChannelID, deleteMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// GET /api/tunnel
func (h *AdminHandler) Tunnel(c *gin.Context) {
	urls := make([]string, 0)
	ctx := c.Request.Context()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:4040/api/tunnels", nil)
	if err == nil {
		client := &http.Client{Timeout: 1 * time.Second}
		if resp, err := client.Do(req); err == nil {
			defer resp.Body.Close()
			var result struct {
				Tunnels []struct {
					PublicURL string `json:"public_url"`
				} `json:"tunnels"`
			}
			if json.NewDecoder(resp.Body).Decode(&result) == nil {
				for _, t := range result.Tunnels {
					if t.PublicURL != "" {
						urls = append(urls, t.PublicURL)
					}
				}
			}
		}
	}

	if len(urls) == 0 {
		if data, err := os.ReadFile(".tunnel_url"); err == nil {
			line := strings.TrimSpace(string(data))
			if line != "" {
				urls = append(urls, line)
			}
		}
	}

	if len(urls) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "没有检测到隧道，请确认 ngrok 或 Cloudflare Tunnel 已启动"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"urls": urls})
}

// POST /api/admin/broadcast {content}
func (h *AdminHandler) Broadcast(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.hub != nil {
		h.hub.BroadcastSystemAll(req.Content)
	}
	c.JSON(http.StatusOK, gin.H{"broadcast": req.Content})
}
