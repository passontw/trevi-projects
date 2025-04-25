package user

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"g38_lottery_service/internal/model"
	userRepo "g38_lottery_service/internal/repository/user"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Service 用戶服務接口
type Service interface {
	Create(ctx context.Context, req model.CreateUserRequest) (*model.UserResponse, error)
	GetByID(ctx context.Context, id uint) (*model.UserResponse, error)
	Update(ctx context.Context, id uint, req model.UpdateUserRequest) (*model.UserResponse, error)
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, page, pageSize int) ([]*model.UserResponse, int64, error)
	Authenticate(ctx context.Context, username, password string) (*model.UserResponse, error)
}

// service 用戶服務實現
type service struct {
	repo  userRepo.Repository
	redis *redis.Client
}

// NewService 創建用戶服務實例
func NewService(repo userRepo.Repository, redis *redis.Client) Service {
	return &service{
		repo:  repo,
		redis: redis,
	}
}

// Create 創建新用戶
func (s *service) Create(ctx context.Context, req model.CreateUserRequest) (*model.UserResponse, error) {
	// 檢查用戶名是否已存在
	existingUser, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// 檢查電子郵件是否已存在
	existingUser, err = s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// 哈希密碼
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 創建用戶
	user := &model.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		FullName:  req.FullName,
		Phone:     req.Phone,
		Address:   req.Address,
		Role:      "user", // 默認角色
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 返回用戶響應（不包含敏感信息）
	response := user.ToResponse()
	return &response, nil
}

// GetByID 根據 ID 獲取用戶
func (s *service) GetByID(ctx context.Context, id uint) (*model.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	response := user.ToResponse()
	return &response, nil
}

// Update 更新用戶信息
func (s *service) Update(ctx context.Context, id uint, req model.UpdateUserRequest) (*model.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// 更新字段（如果提供了）
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Address != "" {
		user.Address = req.Address
	}
	if req.Status != "" {
		user.Status = req.Status
	}

	user.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	response := user.ToResponse()
	return &response, nil
}

// Delete 刪除用戶
func (s *service) Delete(ctx context.Context, id uint) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	return s.repo.Delete(ctx, id)
}

// List 獲取用戶列表（分頁）
func (s *service) List(ctx context.Context, page, pageSize int) ([]*model.UserResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	users, count, err := s.repo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 轉換為響應模型
	var responses []*model.UserResponse
	for _, user := range users {
		response := user.ToResponse()
		responses = append(responses, &response)
	}

	return responses, count, nil
}

// Authenticate 驗證用戶憑證
func (s *service) Authenticate(ctx context.Context, username, password string) (*model.UserResponse, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// 驗證密碼
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 檢查用戶狀態
	if user.Status != "active" {
		return nil, errors.New("user account is not active")
	}

	response := user.ToResponse()
	return &response, nil
}
