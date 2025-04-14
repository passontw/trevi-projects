package service

import (
	"errors"
	"time"

	"g38_lottery_servic/internal/config"
	"g38_lottery_servic/internal/interfaces"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService interface {
	CreateUser(params *CreateUserParams) (*interfaces.User, error)
	GetUsers(page, pageSize int) (*interfaces.UsersResponse, error)
	GetUserById(id uint) (*interfaces.User, error)
	Login(phone, password string) (string, interfaces.User, error)
}

type CreateUserParams struct {
	Password string `json:"password" binding:"required"`
	Phone    string `json:"phone" binding:"required,phone"`
}

type userService struct {
	db     *gorm.DB
	config *config.Config
}

func NewUserService(db *gorm.DB, config *config.Config) UserService {
	return &userService{
		db:     db,
		config: config,
	}
}

func (s *userService) CreateUser(params *CreateUserParams) (*interfaces.User, error) {
	var existingUser interfaces.User

	if err := s.db.Where("phone = ?", params.Phone).First(&existingUser).Error; err == nil {
		return nil, errors.New("手機號碼已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &interfaces.User{
		Password: string(hashedPassword),
		Phone:    params.Phone,
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}
func (s *userService) GetUsers(page, pageSize int) (*interfaces.UsersResponse, error) {
	if page <= 0 {
		page = 1
	}

	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	var users []interfaces.User
	var total int64

	if err := s.db.Model(&interfaces.User{}).Count(&total).Error; err != nil {
		return nil, err
	}

	if err := s.db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, err
	}

	return &interfaces.UsersResponse{
		Users: users,
		Pagination: interfaces.Pagination{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	}, nil
}

func (s *userService) GetUserById(id uint) (*interfaces.User, error) {
	var user interfaces.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用戶不存在")
		}
		return nil, err
	}
	return &user, nil
}

func (s *userService) Login(phone, password string) (string, interfaces.User, error) {
	var user interfaces.User
	if err := s.db.Where("phone = ?", phone).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", interfaces.User{}, errors.New("用戶電話號碼或密碼錯誤")
		}
		return "", interfaces.User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", interfaces.User{}, errors.New("用戶名或密碼錯誤")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(s.config.JWT.ExpiresIn).Unix(),
	})

	signedToken, err := token.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		return "", interfaces.User{}, err
	}

	return signedToken, user, nil
}
