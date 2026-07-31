package repository

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrUserExists = errors.New("user already exists")

type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleAdmin      Role = "admin"
	RoleUser       Role = "user"
)

type User struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)"`
	Username     string    `gorm:"uniqueIndex;type:varchar(64);not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	Role         string    `gorm:"type:varchar(16);default:user;not null"`
	Banned       bool      `gorm:"default:false;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (User) TableName() string {
	return "users"
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return
}

type UserRepository interface {
	GetUserByUsername(username string) (*User, error)
	GetUserByID(id string) (*User, error)
	CreateUser(username, passwordHash string) error
	SetPasswordHash(username, hash string) error
	SetUsername(userID, newUsername string) error
	List() ([]User, error)
	Count() (int64, error)
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
	hash, err := bcrypt.GenerateFromPassword([]byte(getInitialAdminPassword()), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("[FATAL] 初始化管理员密码失败: %v", err)
	}
	now := time.Now()
	r.users["admin"] = &User{
		ID:           "1",
		Username:     "admin",
		PasswordHash: string(hash),
		Role:         string(RoleSuperAdmin),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (r *InMemoryUserRepository) GetUserByUsername(username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[username]
	if !ok {
		return nil, nil
	}
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
	now := time.Now()
	r.users[username] = &User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: passwordHash,
		Role:         string(RoleUser),
		CreatedAt:    now,
		UpdatedAt:    now,
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
	user.UpdatedAt = time.Now()
	return nil
}

func (r *InMemoryUserRepository) SetUsername(userID, newUsername string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Username == newUsername && u.ID != userID {
			return ErrUserExists
		}
	}
	for oldUsername, u := range r.users {
		if u.ID == userID {
			u.Username = newUsername
			u.UpdatedAt = time.Now()
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

func (r *InMemoryUserRepository) Count() (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.users)), nil
}

func (r *InMemoryUserRepository) SetRole(userID, role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.ID == userID {
			u.Role = role
			u.UpdatedAt = time.Now()
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
			u.UpdatedAt = time.Now()
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
	hash, err := bcrypt.GenerateFromPassword([]byte(getInitialAdminPassword()), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("[FATAL] 初始化管理员密码失败: %v", err)
	}

	var user User
	result := r.db.Where("username = ?", "admin").
		Attrs(User{
			PasswordHash: string(hash),
			Role:         string(RoleSuperAdmin),
		}).
		FirstOrCreate(&user)

	if result.RowsAffected == 0 && user.Role != string(RoleSuperAdmin) {
		r.db.Model(&user).Update("role", string(RoleSuperAdmin))
	}
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
	err := r.db.Create(&User{
		Username:     username,
		PasswordHash: passwordHash,
		Role:         string(RoleUser),
	}).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrUserExists
	}
	return err
}

func (r *MySQLUserRepository) SetPasswordHash(username, hash string) error {
	result := r.db.Model(&User{}).Where("username = ?", username).Update("password_hash", hash)
	if result.RowsAffected == 0 {
		return fmt.Errorf("user %q not found", username)
	}
	return result.Error
}

func (r *MySQLUserRepository) SetUsername(userID, newUsername string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := tx.Where("id = ?", userID).First(&user).Error; err != nil {
			return fmt.Errorf("user %q not found", userID)
		}
		oldUsername := user.Username

		err := tx.Model(&User{}).Where("id = ?", userID).Update("username", newUsername).Error
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrUserExists
		} else if err != nil {
			return err
		}

		return tx.Model(&Message{}).Where("username = ?", oldUsername).Update("username", newUsername).Error
	})
}

func (r *MySQLUserRepository) List() ([]User, error) {
	var users []User
	if err := r.db.Order("username ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *MySQLUserRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&User{}).Count(&count).Error
	return count, err
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

func getInitialAdminPassword() string {
	if pwd := os.Getenv("ADMIN_INITIAL_PASSWORD"); pwd != "" {
		return pwd
	}
	log.Println("[WARN] ADMIN_INITIAL_PASSWORD 未设置，使用默认密码 'admin123'。请登录后立即修改！")
	return "admin123"
}
