package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"g38_lottery_service/internal/interfaces"
	"g38_lottery_service/internal/service"
)

// 模擬 AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(phone, password string) (string, interfaces.User, error) {
	args := m.Called(phone, password)
	return args.String(0), args.Get(1).(interfaces.User), args.Error(2)
}

func (m *MockAuthService) Logout(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockAuthService) ValidateToken(token string) (uint, error) {
	args := m.Called(token)
	return uint(args.Int(0)), args.Error(1)
}

// 模擬 UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUserById(id uint) (*interfaces.User, error) {
	args := m.Called(id)
	return args.Get(0).(*interfaces.User), args.Error(1)
}

func (m *MockUserService) GetUsers(page, pageSize int) (*interfaces.UsersResponse, error) {
	args := m.Called(page, pageSize)
	return args.Get(0).(*interfaces.UsersResponse), args.Error(1)
}

func (m *MockUserService) Login(phone, password string) (string, interfaces.User, error) {
	args := m.Called(phone, password)
	return args.String(0), args.Get(1).(interfaces.User), args.Error(2)
}

func (m *MockUserService) CreateUser(params *service.CreateUserParams) (*interfaces.User, error) {
	args := m.Called(params)
	return args.Get(0).(*interfaces.User), args.Error(1)
}

// AuthHandlerTestSuite 定義測試套件
type AuthHandlerTestSuite struct {
	suite.Suite
	router      *gin.Engine
	authService *MockAuthService
	userService *MockUserService
	authHandler *AuthHandler
}

// SetupTest 在每個測試前初始化環境
func (suite *AuthHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.authService = new(MockAuthService)
	suite.userService = new(MockUserService)
	suite.authHandler = NewAuthHandler(suite.authService, suite.userService)

	// 設置路由
	auth := suite.router.Group("/api/v1/auth")
	{
		auth.POST("", suite.authHandler.UserLogin)
		auth.POST("/logout", suite.authHandler.UserLogout)
		auth.GET("/token", suite.authHandler.ValidateToken)
	}
}

// TestAuthHandlerSuite 執行測試套件
func TestAuthHandlerSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerTestSuite))
}

// TestUserLogin 測試登入功能
func (suite *AuthHandlerTestSuite) TestUserLoginSuccess() {
	// 準備測試數據
	loginRequest := interfaces.LoginRequest{
		Phone:    "0987654321",
		Password: "password123",
	}
	expectedToken := "mock.token.123"
	expectedUser := interfaces.User{
		ID:    1,
		Phone: "0987654321",
	}

	// 設定 Mock 行為
	suite.authService.On("Login", loginRequest.Phone, loginRequest.Password).
		Return(expectedToken, expectedUser, nil)

	// 創建請求
	reqBody, _ := json.Marshal(loginRequest)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// 創建響應記錄器
	w := httptest.NewRecorder()

	// 處理請求
	suite.router.ServeHTTP(w, req)

	// 檢查響應
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response interfaces.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedToken, response.Token)
	assert.Equal(suite.T(), expectedUser, response.User)

	// 驗證 Mock 調用
	suite.authService.AssertExpectations(suite.T())
}

// TestUserLoginInvalidCredentials 測試登入失敗 - 無效憑據
func (suite *AuthHandlerTestSuite) TestUserLoginInvalidCredentials() {
	// 準備測試數據
	loginRequest := interfaces.LoginRequest{
		Phone:    "0987654321",
		Password: "wrongpassword",
	}
	expectedError := errors.New("用戶電話號碼或密碼錯誤")

	// 設定 Mock 行為
	suite.authService.On("Login", loginRequest.Phone, loginRequest.Password).
		Return("", interfaces.User{}, expectedError)

	// 創建請求
	reqBody, _ := json.Marshal(loginRequest)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// 創建響應記錄器
	w := httptest.NewRecorder()

	// 處理請求
	suite.router.ServeHTTP(w, req)

	// 檢查響應
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	// 驗證 Mock 調用
	suite.authService.AssertExpectations(suite.T())
}

// TestUserLogout 測試登出功能
func (suite *AuthHandlerTestSuite) TestUserLogoutSuccess() {
	// 準備測試數據
	token := "Bearer mock.token.123"

	// 設定 Mock 行為
	suite.authService.On("Logout", "mock.token.123").Return(nil)

	// 創建請求
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", token)

	// 創建響應記錄器
	w := httptest.NewRecorder()

	// 處理請求
	suite.router.ServeHTTP(w, req)

	// 檢查響應
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// 驗證 Mock 調用
	suite.authService.AssertExpectations(suite.T())
}

// TestUserLogoutNoToken 測試登出失敗 - 無令牌
func (suite *AuthHandlerTestSuite) TestUserLogoutNoToken() {
	// 創建請求（不包含 Authorization 頭）
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)

	// 創建響應記錄器
	w := httptest.NewRecorder()

	// 處理請求
	suite.router.ServeHTTP(w, req)

	// 檢查響應
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestValidateToken 測試令牌驗證功能
func (suite *AuthHandlerTestSuite) TestValidateTokenSuccess() {
	// 重新設置路由，包含中間件處理
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 模擬中間件已經設置了 userID
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})

	// 設置路由
	auth := router.Group("/api/v1/auth")
	{
		auth.GET("/token", suite.authHandler.ValidateToken)
	}

	// 準備測試數據
	expectedUserID := uint(1)
	expectedUser := &interfaces.User{
		ID:    1,
		Phone: "0987654321",
	}

	// 設定 Mock 行為
	suite.userService.On("GetUserById", expectedUserID).Return(expectedUser, nil)

	// 創建請求
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/token", nil)

	// 創建響應記錄器
	w := httptest.NewRecorder()

	// 處理請求
	router.ServeHTTP(w, req)

	// 檢查響應
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// 驗證 Mock 調用
	suite.userService.AssertExpectations(suite.T())
}

// TestValidateTokenUnauthorized 測試令牌驗證失敗 - 未認證
func (suite *AuthHandlerTestSuite) TestValidateTokenUnauthorized() {
	// 創建請求
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/token", nil)

	// 創建響應記錄器
	w := httptest.NewRecorder()

	// 處理請求 - 沒有設置 userID，應該返回未認證錯誤
	suite.router.ServeHTTP(w, req)

	// 檢查響應
	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}
