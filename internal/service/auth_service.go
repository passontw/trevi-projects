package service

import (
	"errors"
	"time"

	"g38_lottery_service/internal/config"
	"g38_lottery_service/internal/interfaces"
	redis "g38_lottery_service/pkg/redisManager"

	"context"

	"github.com/golang-jwt/jwt/v5"
)

type AuthService interface {
	Login(phone, password string) (string, interfaces.User, error)
	Logout(token string) error
	ValidateToken(token string) (uint, error)
}

type authService struct {
	userService UserService
	config      *config.Config
	redisClient redis.RedisManager
}

func NewAuthService(userService UserService, config *config.Config, redisClient redis.RedisManager) AuthService {
	return &authService{
		userService: userService,
		config:      config,
		redisClient: redisClient,
	}
}

func (s *authService) Login(phone, password string) (string, interfaces.User, error) {
	token, user, err := s.userService.Login(phone, password)
	if err != nil {
		return "", interfaces.User{}, err
	}
	return token, user, nil
}

func (s *authService) Logout(token string) error {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWT.Secret), nil
	})

	if err != nil {
		err = s.redisClient.Set(context.Background(), "blacklist:"+token, "true", s.config.JWT.ExpiresIn)
		return err
	}

	if exp, ok := claims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		ttl := time.Until(expTime)
		if ttl > 0 {
			err = s.redisClient.Set(context.Background(), "blacklist:"+token, "true", ttl)
			return err
		}
	}

	err = s.redisClient.Set(context.Background(), "blacklist:"+token, "true", s.config.JWT.ExpiresIn)
	return err
}

func (s *authService) ValidateToken(token string) (uint, error) {
	exists, err := s.redisClient.Exists(context.Background(), "blacklist:"+token)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, errors.New("令牌已被撤銷")
	}

	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWT.Secret), nil
	})

	if err != nil {
		return 0, err
	}

	if exp, ok := claims["exp"].(float64); ok {
		if float64(time.Now().Unix()) > exp {
			return 0, errors.New("令牌已過期")
		}
	}

	if sub, ok := claims["sub"].(float64); ok {
		return uint(sub), nil
	}

	return 0, errors.New("無效的令牌聲明")
}
