package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/pkg/ws"
)

const (
	MaxMessageLength  = 4096 // 单条消息最大字符数
	MaxChannelNameLen = 32   // 频道名称最大字符数
)

type ChatHandler struct {
	channelRepo repository.ChannelRepository
	messageRepo repository.MessageRepository
	hub         *ws.Hub
	bus         ws.MessageBus // 可选，为 nil 则单进程模式
}

func NewChatHandler(channelRepo repository.ChannelRepository, messageRepo repository.MessageRepository, hub *ws.Hub, bus ws.MessageBus) *ChatHandler {
	return &ChatHandler{
		channelRepo: channelRepo,
		messageRepo: messageRepo,
		hub:         hub,
		bus:         bus,
	}
}

type CreateChannelRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *ChatHandler) CreateChannel(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !exists || !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 防御性编程：限制频道名称长度
	if utf8.RuneCountInString(req.Name) > MaxChannelNameLen {
		c.JSON(http.StatusBadRequest, gin.H{"error": "频道名称过长，最多 32 个字符"})
		return
	}

	ch := &repository.Channel{
		Name:      req.Name,
		CreatedBy: userID,
	}

	if h.channelRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "repository not initialized"})
		return
	}

	if err := h.channelRepo.Create(ch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建频道失败"})
		return
	}

	// 广播频道创建事件给所有在线用户
	if h.hub != nil {
		if chMsg, err := json.Marshal(ws.WSMessage{
			Type:      "channel_created",
			ChannelID: ch.ID,
			Content:   ch.Name,
			CreatedAt: ch.CreatedAt.Format(time.RFC3339),
		}); err == nil {
			h.hub.Broadcast(chMsg)
		}
	}

	c.JSON(http.StatusCreated, ch)
}

func (h *ChatHandler) ListChannels(c *gin.Context) {
	if h.channelRepo == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	channels, err := h.channelRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取频道列表失败"})
		return
	}
	c.JSON(http.StatusOK, channels)
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	channelID := c.Param("id")
	limitStr := c.DefaultQuery("limit", "50")
	limit, limitErr := strconv.Atoi(limitStr)
	if limitErr != nil || limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	before := c.Query("before")

	if h.messageRepo == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	var msgs []repository.Message
	var err error
	if before != "" {
		t, parseErr := time.Parse(time.RFC3339, before)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "分页参数格式错误"})
			return
		}
		msgs, err = h.messageRepo.GetByChannelBefore(channelID, t, limit)
	} else {
		msgs, err = h.messageRepo.GetByChannel(channelID, limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息失败"})
		return
	}
	c.JSON(http.StatusOK, msgs)
}

func (h *ChatHandler) EditMessage(c *gin.Context) {
	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	msgID := c.Param("id")

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if utf8.RuneCountInString(req.Content) > MaxMessageLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息内容过长，最多 4096 个字符"})
		return
	}

	if h.messageRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "repository not initialized"})
		return
	}

	msg, err := h.messageRepo.GetByID(msgID)
	if err != nil || msg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
		return
	}
	if msg.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能编辑自己的消息"})
		return
	}

	if err := h.messageRepo.Update(msgID, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "编辑消息失败"})
		return
	}

	if h.hub != nil {
		if editMsg, err := json.Marshal(ws.WSMessage{
			Type:      "message_edited",
			ChannelID: msg.ChannelID,
			UserID:    userID,
			Content:   req.Content,
		}); err == nil {
			h.hub.BroadcastToChannel(msg.ChannelID, editMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "编辑成功"})
}

func (h *ChatHandler) DeleteMyMessage(c *gin.Context) {
	userIDVal, _ := c.Get("user_id")
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	msgID := c.Param("id")

	if h.messageRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "repository not initialized"})
		return
	}

	msg, err := h.messageRepo.GetByID(msgID)
	if err != nil || msg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
		return
	}

	if err := h.messageRepo.DeleteByUser(msgID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能删除自己的消息"})
		return
	}

	if h.hub != nil {
		if deleteMsg, err := json.Marshal(ws.WSMessage{
			Type:      "message_deleted",
			ChannelID: msg.ChannelID,
			Content:   msgID,
		}); err == nil {
			h.hub.BroadcastToChannel(msg.ChannelID, deleteMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// OnWSMessage — 先同步持久化获取消息 ID，再广播 (带有 3s DB 探活超时防护)
func (h *ChatHandler) OnWSMessage(client *ws.Client, data []byte) {
	var msg ws.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	// 消息长度限制，防止内存耗尽攻击
	if utf8.RuneCountInString(msg.Content) > MaxMessageLength {
		errMsg, _ := json.Marshal(ws.WSMessage{
			Type:    "error",
			Content: "消息内容过长，最多 4096 个字符",
		})
		client.SendMessage(errMsg)
		return
	}

	if !client.IsInChannel(msg.ChannelID) {
		errMsg, _ := json.Marshal(ws.WSMessage{
			Type:    "error",
			Content: "你未加入此频道",
		})
		client.SendMessage(errMsg)
		return
	}

	dbMsg := &repository.Message{
		ChannelID: msg.ChannelID,
		UserID:    client.GetUserID(),
		Username:  client.GetUsername(),
		Content:   msg.Content,
	}

	// 核心防护：设置 3s 超时防慢查询卡死 WS 读循环
	done := make(chan error, 1)
	go func() {
		if h.messageRepo != nil {
			done <- h.messageRepo.Create(dbMsg)
		} else {
			done <- nil
		}
	}()

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("[WARN] 消息持久化失败: %v (channel=%s, user=%s)", err, msg.ChannelID, client.GetUsername())
			errMsg, _ := json.Marshal(ws.WSMessage{
				Type:    "error",
				Content: "发送失败，服务器繁忙",
			})
			client.SendMessage(errMsg)
			return
		}
	case <-timer.C:
		log.Printf("[WARN] 消息持久化超时 (channel=%s, user=%s)", msg.ChannelID, client.GetUsername())
		errMsg, _ := json.Marshal(ws.WSMessage{
			Type:    "error",
			Content: "发送超时，请重试",
		})
		client.SendMessage(errMsg)
		return
	}

	now := time.Now().Format(time.RFC3339)

	outMsg, _ := json.Marshal(ws.WSMessage{
		Type:      "message",
		ChannelID: msg.ChannelID,
		UserID:    client.GetUserID(),
		Username:  client.GetUsername(),
		Content:   msg.Content,
		CreatedAt: now,
		MessageID: dbMsg.ID,
	})

	if h.bus != nil {
		pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := h.bus.Publish(pubCtx, msg.ChannelID, outMsg); err != nil {
			log.Printf("[WARN] 发布消息到 Redis 消息总线失败: %v (channel=%s)", err, msg.ChannelID)
		}
		cancel()
	} else if h.hub != nil {
		h.hub.BroadcastToChannel(msg.ChannelID, outMsg)
	}
}
