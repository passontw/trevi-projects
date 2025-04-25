package model

import (
	"time"

	"gorm.io/gorm"
)

// User 使用者模型
type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Username  string         `gorm:"size:50;not null;uniqueIndex" json:"username"`
	Email     string         `gorm:"size:100;not null;uniqueIndex" json:"email"`
	Password  string         `gorm:"size:100;not null" json:"-"` // 密碼不返回給前端
	FullName  string         `gorm:"size:100" json:"full_name"`
	Phone     string         `gorm:"size:20" json:"phone"`
	Address   string         `gorm:"size:255" json:"address"`
	Role      string         `gorm:"size:20;default:'user'" json:"role"` // user, admin, merchant
	Status    string         `gorm:"size:20;default:'active'" json:"status"` // active, inactive, suspended
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定資料表名稱
func (User) TableName() string {
	return "users"
}

// UserResponse 用戶響應模型（不包含敏感信息）
type UserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone,omitempty"`
	Address   string    `json:"address,omitempty"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse 將完整用戶模型轉換為安全的響應模型
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		FullName:  u.FullName,
		Phone:     u.Phone,
		Address:   u.Address,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// CreateUserRequest 創建用戶請求模型
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone" binding:"omitempty"`
	Address  string `json:"address" binding:"omitempty"`
}

// UpdateUserRequest 更新用戶請求模型
type UpdateUserRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	FullName string `json:"full_name" binding:"omitempty"`
	Phone    string `json:"phone" binding:"omitempty"`
	Address  string `json:"address" binding:"omitempty"`
	Status   string `json:"status" binding:"omitempty,oneof=active inactive suspended"`
}
