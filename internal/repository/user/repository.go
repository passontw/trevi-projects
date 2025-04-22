package user

import (
	"context"
	"errors"

	"g38_lottery_service/internal/model"

	"gorm.io/gorm"
)

// Repository 用戶倉庫接口
type Repository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, offset, limit int) ([]*model.User, int64, error)
}

// repository 使用 GORM 實現 Repository 接口
type repository struct {
	db *gorm.DB
}

// NewRepository 創建用戶倉庫實例
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create 創建新用戶
func (r *repository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID 根據 ID 獲取用戶
func (r *repository) GetByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 用戶不存在但不是錯誤
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根據用戶名獲取用戶
func (r *repository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根據電子郵件獲取用戶
func (r *repository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update 更新用戶信息
func (r *repository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete 刪除用戶
func (r *repository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}

// List 獲取用戶列表（分頁）
func (r *repository) List(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
	var users []*model.User
	var count int64

	// 獲取總數
	if err := r.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	// 獲取分頁數據
	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, count, nil
}
