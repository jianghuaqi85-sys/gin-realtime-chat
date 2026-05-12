package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

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
	users, err := h.userRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户统计失败"})
		return
	}
	chCount, err := h.channelRepo.Count()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取频道统计失败"})
		return
	}
	msgCount, err := h.messageRepo.Count()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息统计失败"})
		return
	}
	online := h.hub.OnlineCount()

	c.JSON(http.StatusOK, gin.H{
		"users_total": len(users),
		"online":      online,
		"channels":    chCount,
		"messages":    msgCount,
	})
}

// GET /api/admin/users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.userRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}
	// 隐藏密码哈希
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

	// 先断开连接
	h.hub.DisconnectUser(userID)

	// 再删除账号
	if err := h.userRepo.Delete(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": userID})
}

// POST /api/admin/ban  {user_id}
func (h *AdminHandler) Ban(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.userRepo.SetBanned(req.UserID, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "封禁用户失败"})
		return
	}

	// 广播用户封禁事件给所有在线用户
	banMsg, _ := json.Marshal(ws.WSMessage{
		Type:    "user_banned",
		UserID:  req.UserID,
		Content: req.UserID,
	})
	h.hub.Broadcast(banMsg)

	// 断开被封禁用户的连接（在广播之后，确保用户能收到通知）
	h.hub.DisconnectUser(req.UserID)

	c.JSON(http.StatusOK, gin.H{"banned": req.UserID})
}

// POST /api/admin/unban  {user_id}
func (h *AdminHandler) Unban(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.userRepo.SetBanned(req.UserID, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解封用户失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unbanned": req.UserID})
}

// DELETE /api/admin/channels/:id
func (h *AdminHandler) DeleteChannel(c *gin.Context) {
	id := c.Param("id")
	if err := h.channelRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除频道失败"})
		return
	}

	// 广播频道删除事件给所有在线用户
	deleteMsg, _ := json.Marshal(ws.WSMessage{
		Type:      "channel_deleted",
		ChannelID: id,
	})
	h.hub.Broadcast(deleteMsg)

	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// DELETE /api/admin/channels/:id/messages
func (h *AdminHandler) ClearMessages(c *gin.Context) {
	channelID := c.Param("id")
	if err := h.messageRepo.DeleteByChannel(channelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清空消息失败"})
		return
	}

	// 广播清空消息事件到该频道
	clearMsg, _ := json.Marshal(ws.WSMessage{
		Type:      "messages_cleared",
		ChannelID: channelID,
	})
	h.hub.BroadcastToChannel(channelID, clearMsg)

	c.JSON(http.StatusOK, gin.H{"cleared": channelID})
}

// DELETE /api/admin/messages/:id
func (h *AdminHandler) DeleteMessage(c *gin.Context) {
	id := c.Param("id")

	// 先获取消息信息，以便广播到正确的频道
	msg, err := h.messageRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
		return
	}

	if err := h.messageRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除消息失败"})
		return
	}

	// 广播消息删除事件到消息所属频道
	deleteMsg, _ := json.Marshal(ws.WSMessage{
		Type:      "message_deleted",
		ChannelID: msg.ChannelID,
		Content:   id,
	})
	h.hub.BroadcastToChannel(msg.ChannelID, deleteMsg)

	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// GET /api/tunnel — 获取公网地址（支持 ngrok 和 Cloudflare Tunnel）
func (h *AdminHandler) Tunnel(c *gin.Context) {
	urls := make([]string, 0)

	// 方式 1：尝试 ngrok 本地 API
	client := &http.Client{Timeout: 1 * time.Second}
	if resp, err := client.Get("http://127.0.0.1:4040/api/tunnels"); err == nil {
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

	// 方式 2：尝试读取 Cloudflare Tunnel URL 文件
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

// POST /api/admin/broadcast  {content}
func (h *AdminHandler) Broadcast(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.hub.BroadcastSystemAll(req.Content)
	c.JSON(http.StatusOK, gin.H{"broadcast": req.Content})
}
