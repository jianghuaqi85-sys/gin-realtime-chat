package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/pkg/ws"
)

const MaxMessageLength = 200

type ChatHandler struct {
	channelRepo repository.ChannelRepository
	messageRepo repository.MessageRepository
	hub         *ws.Hub
	bus         ws.MessageBus // 可选，为 nil 则单进程模式
	persistCh   chan *repository.Message
}

func NewChatHandler(channelRepo repository.ChannelRepository, messageRepo repository.MessageRepository, hub *ws.Hub, bus ws.MessageBus) *ChatHandler {
	h := &ChatHandler{
		channelRepo: channelRepo,
		messageRepo: messageRepo,
		hub:         hub,
		bus:         bus,
		persistCh:   make(chan *repository.Message, 4096),
	}
	// 启动持久化 worker，数量等于 CPU 核数
	workers := runtime.NumCPU()
	if workers < 2 {
		workers = 2
	}
	for i := 0; i < workers; i++ {
		go h.persistWorker()
	}
	return h
}

func (h *ChatHandler) persistWorker() {
	for msg := range h.persistCh {
		if err := h.messageRepo.Create(msg); err != nil {
			log.Printf("[WARN] 消息持久化失败: %v (channel=%s, user=%s)", err, msg.ChannelID, msg.Username)
		}
	}
}

type CreateChannelRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *ChatHandler) CreateChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch := &repository.Channel{
		Name:      req.Name,
		CreatedBy: userID,
	}
	if err := h.channelRepo.Create(ch); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建频道失败"})
		return
	}

	// 广播频道创建事件给所有在线用户
	chMsg, _ := json.Marshal(ws.WSMessage{
		Type:      "channel_created",
		ChannelID: ch.ID,
		Content:   ch.Name,
		CreatedAt: ch.CreatedAt.Format(time.RFC3339),
	})
	h.hub.Broadcast(chMsg)

	c.JSON(http.StatusCreated, ch)
}

func (h *ChatHandler) ListChannels(c *gin.Context) {
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
	limit, _ := strconv.Atoi(limitStr)
	before := c.Query("before")

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
	userID := c.MustGet("user_id").(string)
	msgID := c.Param("id")

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 先获取消息信息，以便广播到正确的频道
	msg, err := h.messageRepo.GetByID(msgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
		return
	}

	if err := h.messageRepo.Update(msgID, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "编辑消息失败"})
		return
	}

	// 广播消息编辑事件到消息所属频道
	editMsg, _ := json.Marshal(ws.WSMessage{
		Type:      "message_edited",
		ChannelID: msg.ChannelID,
		UserID:    userID,
		Content:   req.Content,
	})
	h.hub.BroadcastToChannel(msg.ChannelID, editMsg)

	_ = userID
	c.JSON(http.StatusOK, gin.H{"message": "编辑成功"})
}

func (h *ChatHandler) DeleteMyMessage(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	msgID := c.Param("id")
	log.Printf("[DEBUG] DeleteMyMessage: msgID=%s, userID=%s", msgID, userID)

	// 先获取消息信息，以便广播到正确的频道
	msg, err := h.messageRepo.GetByID(msgID)
	if err != nil {
		log.Printf("[DEBUG] GetByID failed: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
		return
	}
	log.Printf("[DEBUG] GetByID success: msg.UserID=%s", msg.UserID)

	if err := h.messageRepo.DeleteByUser(msgID, userID); err != nil {
		log.Printf("[DEBUG] DeleteByUser failed: %v", err)
		c.JSON(http.StatusForbidden, gin.H{"error": "只能删除自己的消息"})
		return
	}
	log.Printf("[DEBUG] DeleteByUser success")

	// 广播消息删除事件到消息所属频道
	deleteMsg, _ := json.Marshal(ws.WSMessage{
		Type:      "message_deleted",
		ChannelID: msg.ChannelID,
		Content:   msgID,
	})
	h.hub.BroadcastToChannel(msg.ChannelID, deleteMsg)

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// OnWSMessage — 先同步持久化获取消息 ID，再广播
func (h *ChatHandler) OnWSMessage(client *ws.Client, data []byte) {
	var msg ws.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	// 消息长度限制，防止内存耗尽攻击
	if len(msg.Content) > MaxMessageLength {
		errMsg, _ := json.Marshal(ws.WSMessage{
			Type:    "error",
			Content: "消息内容过长，最多 200 个字符",
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

	// 先同步持久化到数据库，获取消息 ID
	dbMsg := &repository.Message{
		ChannelID: msg.ChannelID,
		UserID:    client.GetUserID(),
		Username:  client.GetUsername(),
		Content:   msg.Content,
	}
	if err := h.messageRepo.Create(dbMsg); err != nil {
		log.Printf("[WARN] 消息持久化失败: %v (channel=%s, user=%s)", err, msg.ChannelID, client.GetUsername())
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

	// 通过 Redis Pub/Sub 发布（多实例广播）或直接本地广播（单实例）
	if h.bus != nil {
		h.bus.Publish(msg.ChannelID, outMsg)
	} else {
		h.hub.BroadcastToChannel(msg.ChannelID, outMsg)
	}
}
