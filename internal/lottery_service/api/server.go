package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"g38_lottery_service/internal/lottery_service/config"
	"g38_lottery_service/internal/lottery_service/gameflow"
	"g38_lottery_service/internal/lottery_service/grpc/dealer"
	"g38_lottery_service/pkg/healthcheck"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// APIServer 處理 API 請求的 HTTP 服務器
type APIServer struct {
	config        *config.AppConfig
	logger        *zap.Logger
	gameManager   *gameflow.GameManager
	repository    *gameflow.CompositeRepository
	server        *http.Server
	health        *healthcheck.Manager
	dealerAdapter *dealer.DealerServiceAdapter
}

// NewAPIServer 創建新的 API 服務器
func NewAPIServer(
	config *config.AppConfig,
	logger *zap.Logger,
	gameManager *gameflow.GameManager,
	repository *gameflow.CompositeRepository,
	health *healthcheck.Manager,
	dealerAdapter *dealer.DealerServiceAdapter,
) *APIServer {
	// 確保 logger 有適當的 tag
	apiLogger := logger.With(zap.String("component", "api_server"))

	// 創建 API 服務器
	server := &APIServer{
		config:        config,
		logger:        apiLogger,
		gameManager:   gameManager,
		repository:    repository,
		health:        health,
		dealerAdapter: dealerAdapter,
	}

	return server
}

// Start 啟動 API 服務器
func (s *APIServer) Start(lc fx.Lifecycle) {
	// 使用應用配置中的 API 端口
	port := s.config.Server.Port
	host := s.config.Server.Host
	serverAddr := fmt.Sprintf("%s:%d", host, port)
	s.logger.Info("正在啟動 API 服務器",
		zap.String("address", serverAddr),
		zap.Int("port", port))

	// 建立 ServeMux
	mux := http.NewServeMux()

	// 註冊 API 路由
	s.registerRoutes(mux)

	// 安裝健康檢查路由
	s.health.InstallHandlers(mux)

	// 添加遊戲管理器健康檢查
	s.health.AddReadinessCheck(&gameflow.GameManagerChecker{
		Name_:       "game-manager",
		GameManager: s.gameManager,
	})

	// 如果將來需要添加數據庫健康檢查，可以在這裡添加
	// s.health.AddReadinessCheck(&healthcheck.DatabaseChecker{
	//     Name_: "main-db",
	//     DB:    yourDatabaseConnection,
	// })

	// 建立 HTTP 服務器
	s.server = &http.Server{
		Addr:    serverAddr,
		Handler: mux,
	}

	// 生命周期鉤子
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 標記服務為未就緒
			s.health.SetReady(false)

			// 啟動 HTTP 服務器
			go func() {
				s.logger.Info("API 服務器已啟動",
					zap.String("監聽地址", serverAddr),
					zap.Int("端口", port))

				if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					s.logger.Error("API 服務器運行失敗", zap.Error(err))
				}
			}()

			// 執行初始化任務
			go func() {
				// 在這裡可以添加初始化代碼
				// 例如：確保數據庫連接、檢查外部服務等

				// 假設初始化需要一些時間
				time.Sleep(500 * time.Millisecond)

				// 初始化完成後，標記服務為就緒
				s.logger.Info("API 服務器初始化完成，標記為就緒")
				s.health.SetReady(true)
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			s.logger.Info("正在停止 API 服務器")

			// 標記為未就緒
			s.health.SetReady(false)
			s.logger.Info("API 服務器已標記為未就緒，拒絕新請求")

			// 設置關閉超時時間
			timeout := 30 * time.Second
			if s.config.Server.ShutdownTimeoutSeconds > 0 {
				timeout = time.Duration(s.config.Server.ShutdownTimeoutSeconds) * time.Second
			}

			// 創建帶超時的上下文
			shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// 嘗試優雅關閉
			if err := s.server.Shutdown(shutdownCtx); err != nil {
				s.logger.Error("API 服務器優雅關閉失敗", zap.Error(err))
				return err
			}

			s.logger.Info("API 服務器已優雅關閉")
			return nil
		},
	})
}

// registerRoutes 註冊所有 API 路由
func (s *APIServer) registerRoutes(mux *http.ServeMux) {
	// 健康檢查由 health 包處理，這裡不再需要獨立的健康檢查路由

	// 根路徑
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Lottery Service API Server is running."))
	})

	// 添加自定義抽球 API 路由
	mux.HandleFunc("POST /api/v1/dealer/custom-draw-ball", s.handleCustomDrawBall)

	s.logger.Info("API 路由註冊完成")
}

// handleCustomDrawBall 處理自定義抽球請求
func (s *APIServer) handleCustomDrawBall(w http.ResponseWriter, r *http.Request) {
	// 設置響應類型
	w.Header().Set("Content-Type", "application/json")

	// 解析請求 JSON 數據
	var requestData map[string]interface{}
	if err := decodeJSON(r, &requestData); err != nil {
		s.logger.Error("解析 JSON 請求失敗", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "無效的請求格式")
		return
	}

	// 設置請求上下文，可以包含超時
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 取得 grpc 服務適配器
	dealerAdapter := getDealerAdapter(s)
	if dealerAdapter == nil {
		s.logger.Error("無法獲取 Dealer 服務適配器")
		respondWithError(w, http.StatusInternalServerError, "服務當前不可用")
		return
	}

	// 調用自定義抽球方法
	resp, err := dealerAdapter.CustomDrawBall(ctx, requestData)
	if err != nil {
		s.logger.Error("處理自定義抽球請求失敗", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 返回成功響應
	respondWithJSON(w, http.StatusOK, resp)
}

// decodeJSON 解析 JSON 請求
func decodeJSON(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(v); err != nil {
		return err
	}
	return nil
}

// respondWithError 發送錯誤響應
func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithJSON(w, statusCode, map[string]string{"error": message})
}

// respondWithJSON 發送 JSON 響應
func respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	// 設置響應狀態碼
	w.WriteHeader(statusCode)

	// 如果數據不是 nil，則進行 JSON 編碼
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// 如果編碼失敗，寫入一個簡單的錯誤信息
			w.Write([]byte(`{"error":"JSON 編碼失敗"}`))
		}
	}
}

// getDealerAdapter 獲取 DealerServiceAdapter 實例
func getDealerAdapter(s *APIServer) *dealer.DealerServiceAdapter {
	return s.dealerAdapter
}

// Module 提供 FX 模塊
var Module = fx.Options(
	fx.Provide(NewAPIServer),
	fx.Invoke(func(server *APIServer, lc fx.Lifecycle) {
		server.Start(lc)
	}),
	healthcheck.Module, // 加入健康檢查模塊
)
