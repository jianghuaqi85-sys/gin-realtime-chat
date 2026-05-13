package repository

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrUserExists = errors.New("user already exists")

type User struct {
	ID           string `gorm:"primaryKey;type:varchar(36)"`
	Username     string `gorm:"uniqueIndex;type:varchar(64);not null"`
	PasswordHash string `gorm:"type:varchar(255);not null"`
	Role         string `gorm:"type:varchar(16);default:user;not null"` // admin / user
	Banned       bool   `gorm:"default:false;not null"`
}

func (User) TableName() string {
	return "users"
}

type UserRepository interface {
	GetUserByUsername(username string) (*User, error)
	GetUserByID(id string) (*User, error)
	CreateUser(username, passwordHash string) error
	SetPasswordHash(username, hash string) error
	SetUsername(userID, newUsername string) error
	List() ([]User, error)
	SetRole(userID, role string) error
	SetBanned(userID string, banned bool) error
	Delete(userID string) error
}

// InMemoryUserRepository

type InMemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]*User
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	repo := &InMemoryUserRepository{
		users: make(map[string]*User),
	}
	repo.initDefaultUser()
	return repo
}

func (r *InMemoryUserRepository) initDefaultUser() {
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	r.users["admin"] = &User{
		ID:           "1",
		Username:     "admin",
		PasswordHash: string(hash),
		Role:         "admin",
	}
}

func (r *InMemoryUserRepository) GetUserByUsername(username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[username]
	if !ok {
		return nil, nil
	}
	// 返回深拷贝，避免并发修改内部状态
	cp := *user
	return &cp, nil
}

func (r *InMemoryUserRepository) GetUserByID(id string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.ID == id {
			cp := *u
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *InMemoryUserRepository) CreateUser(username, passwordHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[username]; ok {
		return ErrUserExists
	}
	r.users[username] = &User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: passwordHash,
		Role:         "user",
	}
	return nil
}

func (r *InMemoryUserRepository) SetPasswordHash(username, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[username]
	if !ok {
		return fmt.Errorf("user %q not found", username)
	}
	user.PasswordHash = hash
	return nil
}

func (r *InMemoryUserRepository) SetUsername(userID, newUsername string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// 检查新用户名是否已存在
	for _, u := range r.users {
		if u.Username == newUsername && u.ID != userID {
			return ErrUserExists
		}
	}
	// 找到用户并更新用户名
	for oldUsername, u := range r.users {
		if u.ID == userID {
			u.Username = newUsername
			r.users[newUsername] = u
			delete(r.users, oldUsername)
			return nil
		}
	}
	return fmt.Errorf("user %q not found", userID)
}

func (r *InMemoryUserRepository) List() ([]User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var users []User
	for _, u := range r.users {
		users = append(users, *u)
	}
	return users, nil
}

func (r *InMemoryUserRepository) SetRole(userID, role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.ID == userID {
			u.Role = role
			return nil
		}
	}
	return fmt.Errorf("user %q not found", userID)
}

func (r *InMemoryUserRepository) SetBanned(userID string, banned bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.ID == userID {
			u.Banned = banned
			return nil
		}
	}
	return fmt.Errorf("user %q not found", userID)
}

func (r *InMemoryUserRepository) Delete(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, u := range r.users {
		if u.ID == userID {
			delete(r.users, name)
			return nil
		}
	}
	return fmt.Errorf("user %q not found", userID)
}

// MySQLUserRepository

type MySQLUserRepository struct {
	db *gorm.DB
}

func NewMySQLUserRepository(db *gorm.DB) *MySQLUserRepository {
	repo := &MySQLUserRepository{db: db}
	repo.initDefaultUser()
	return repo
}

func (r *MySQLUserRepository) initDefaultUser() {
	var count int64
	r.db.Model(&User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		r.db.Model(&User{}).Where("username = ?", "admin").Update("role", "admin")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	r.db.Create(&User{
		ID:           uuid.New().String(),
		Username:     "admin",
		PasswordHash: string(hash),
		Role:         "admin",
	})
}

func (r *MySQLUserRepository) GetUserByUsername(username string) (*User, error) {
	var user User
	err := r.db.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *MySQLUserRepository) GetUserByID(id string) (*User, error) {
	var user User
	err := r.db.Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *MySQLUserRepository) CreateUser(username, passwordHash string) error {
	var count int64
	r.db.Model(&User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return ErrUserExists
	}
	return r.db.Create(&User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: passwordHash,
		Role:         "user",
	}).Error
}

func (r *MySQLUserRepository) SetPasswordHash(username, hash string) error {
	result := r.db.Model(&User{}).Where("username = ?", username).Update("password_hash", hash)
	if result.RowsAffected == 0 {
		return fmt.Errorf("user %q not found", username)
	}
	return result.Error
}

func (r *MySQLUserRepository) SetUsername(userID, newUsername string) error {
	// 检查新用户名是否已存在
	var count int64
	r.db.Model(&User{}).Where("username = ? AND id != ?", newUsername, userID).Count(&count)
	if count > 0 {
		return ErrUserExists
	}
	// 获取旧用户名
	var user User
	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("user %q not found", userID)
	}
	oldUsername := user.Username
	// 更新用户表中的用户名
	result := r.db.Model(&User{}).Where("id = ?", userID).Update("username", newUsername)
	if result.Error != nil {
		return result.Error
	}
	// 更新所有历史消息中的用户名
	r.db.Model(&Message{}).Where("username = ?", oldUsername).Update("username", newUsername)
	return nil
}

func (r *MySQLUserRepository) List() ([]User, error) {
	var users []User
	if err := r.db.Order("username ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *MySQLUserRepository) SetRole(userID, role string) error {
	result := r.db.Model(&User{}).Where("id = ?", userID).Update("role", role)
	if result.RowsAffected == 0 {
		return fmt.Errorf("user %q not found", userID)
	}
	return result.Error
}

func (r *MySQLUserRepository) SetBanned(userID string, banned bool) error {
	result := r.db.Model(&User{}).Where("id = ?", userID).Update("banned", banned)
	if result.RowsAffected == 0 {
		return fmt.Errorf("user %q not found", userID)
	}
	return result.Error
}

func (r *MySQLUserRepository) Delete(userID string) error {
	result := r.db.Where("id = ?", userID).Delete(&User{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("user %q not found", userID)
	}
	return result.Error
}
