package repository

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Channel — 聊天频道

type Channel struct {
	ID        string `gorm:"primaryKey;type:varchar(36)"`
	Name      string `gorm:"uniqueIndex;type:varchar(64);not null"`
	CreatedBy string `gorm:"type:varchar(36);not null"`
	CreatedAt time.Time
}

func (Channel) TableName() string { return "channels" }

func (c *Channel) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return
}

type ChannelRepository interface {
	Create(channel *Channel) error
	GetByID(id string) (*Channel, error)
	List() ([]Channel, error)
	Delete(id string) error
	Count() (int64, error)
}

type MySQLChannelRepository struct {
	db *gorm.DB
}

func NewMySQLChannelRepository(db *gorm.DB) *MySQLChannelRepository {
	return &MySQLChannelRepository{db: db}
}

func (r *MySQLChannelRepository) Create(ch *Channel) error {
	return r.db.Create(ch).Error
}

func (r *MySQLChannelRepository) GetByID(id string) (*Channel, error) {
	var ch Channel
	if err := r.db.Where("id = ?", id).First(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *MySQLChannelRepository) List() ([]Channel, error) {
	var channels []Channel
	if err := r.db.Order("created_at ASC").Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *MySQLChannelRepository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("channel_id = ?", id).Delete(&Message{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&Channel{}).Error
	})
}

func (r *MySQLChannelRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&Channel{}).Count(&count).Error
	return count, err
}

// Message — 聊天消息 (包含联合索引 idx_channel_created 与 外键级联描述)

type Message struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)"`
	ChannelID string    `gorm:"index:idx_channel_created;type:varchar(36);not null;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID    string    `gorm:"type:varchar(36);not null"`
	Username  string    `gorm:"type:varchar(64);not null"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"index:idx_channel_created"`
	UpdatedAt time.Time
}

func (Message) TableName() string { return "messages" }

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return
}

type MessageRepository interface {
	Create(msg *Message) error
	GetByID(id string) (*Message, error)
	GetByChannel(channelID string, limit int) ([]Message, error)
	GetByChannelBefore(channelID string, before time.Time, limit int) ([]Message, error)
	Update(id, content string) error
	Delete(id string) error
	DeleteByChannel(channelID string) error
	DeleteByUser(id, userID string) error
	Count() (int64, error)
}

type MySQLMessageRepository struct {
	db *gorm.DB
}

func NewMySQLMessageRepository(db *gorm.DB) *MySQLMessageRepository {
	return &MySQLMessageRepository{db: db}
}

func (r *MySQLMessageRepository) Create(msg *Message) error {
	return r.db.Create(msg).Error
}

func (r *MySQLMessageRepository) GetByChannel(channelID string, limit int) ([]Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var msgs []Message
	if err := r.db.Where("channel_id = ?", channelID).
		Order("created_at DESC").
		Limit(limit).
		Find(&msgs).Error; err != nil {
		return nil, err
	}
	slices.Reverse(msgs)
	return msgs, nil
}

func (r *MySQLMessageRepository) GetByChannelBefore(channelID string, before time.Time, limit int) ([]Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var msgs []Message
	if err := r.db.Where("channel_id = ? AND created_at < ?", channelID, before).
		Order("created_at DESC").
		Limit(limit).
		Find(&msgs).Error; err != nil {
		return nil, err
	}
	slices.Reverse(msgs)
	return msgs, nil
}

func (r *MySQLMessageRepository) GetByID(id string) (*Message, error) {
	var msg Message
	if err := r.db.Where("id = ?", id).First(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *MySQLMessageRepository) Update(id, content string) error {
	result := r.db.Model(&Message{}).Where("id = ?", id).Update("content", content)
	if result.RowsAffected == 0 {
		return fmt.Errorf("message %q not found", id)
	}
	return result.Error
}

func (r *MySQLMessageRepository) DeleteByUser(id, userID string) error {
	result := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&Message{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found or unauthorized")
	}
	return nil
}

func (r *MySQLMessageRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&Message{}).Error
}

func (r *MySQLMessageRepository) DeleteByChannel(channelID string) error {
	return r.db.Where("channel_id = ?", channelID).Delete(&Message{}).Error
}

func (r *MySQLMessageRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&Message{}).Count(&count).Error
	return count, err
}
