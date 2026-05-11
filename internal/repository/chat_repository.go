package repository

import (
	"fmt"
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
	if ch.ID == "" {
		ch.ID = uuid.New().String()
	}
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

// Message — 聊天消息

type Message struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)"`
	ChannelID string    `gorm:"index;type:varchar(36);not null"`
	UserID    string    `gorm:"type:varchar(36);not null"`
	Username  string    `gorm:"type:varchar(64);not null"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"index"`
}

func (Message) TableName() string { return "messages" }

type MessageRepository interface {
	Create(msg *Message) error
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
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
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
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
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
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
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
	if result.RowsAffected == 0 {
		return fmt.Errorf("message not found or not owned by user")
	}
	return result.Error
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
