package gameflow

import (
	"fmt"
)

// GameFlowError 代表遊戲流程錯誤
type GameFlowError struct {
	Code    string
	Message string
}

// Error 實現error接口
func (e *GameFlowError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Is 實現errors.Is接口，用於錯誤比較
func (e *GameFlowError) Is(target error) bool {
	t, ok := target.(*GameFlowError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// 預定義錯誤
var (
	// 階段錯誤
	ErrInvalidStage = &GameFlowError{
		Code:    "INVALID_STAGE",
		Message: "當前階段不允許此操作",
	}

	ErrStageTransitionFailed = &GameFlowError{
		Code:    "STAGE_TRANSITION_FAILED",
		Message: "階段轉換失敗",
	}

	// 球錯誤
	ErrInvalidBall = &GameFlowError{
		Code:    "INVALID_BALL",
		Message: "無效的球號",
	}

	ErrDuplicateBall = &GameFlowError{
		Code:    "DUPLICATE_BALL",
		Message: "重複的球號",
	}

	ErrMaxBallsReached = &GameFlowError{
		Code:    "MAX_BALLS_REACHED",
		Message: "已達到最大球數",
	}

	// 遊戲錯誤
	ErrGameNotFound = &GameFlowError{
		Code:    "GAME_NOT_FOUND",
		Message: "找不到遊戲",
	}

	ErrGameAlreadyExists = &GameFlowError{
		Code:    "GAME_ALREADY_EXISTS",
		Message: "遊戲已存在",
	}

	// 取消錯誤
	ErrCannotCancelGame = &GameFlowError{
		Code:    "CANNOT_CANCEL_GAME",
		Message: "此階段不允許取消遊戲",
	}

	ErrGameAlreadyCancelled = &GameFlowError{
		Code:    "GAME_ALREADY_CANCELLED",
		Message: "遊戲已取消",
	}

	// 持久化錯誤
	ErrPersistenceFailed = &GameFlowError{
		Code:    "PERSISTENCE_FAILED",
		Message: "資料持久化失敗",
	}

	ErrDataCorrupted = &GameFlowError{
		Code:    "DATA_CORRUPTED",
		Message: "資料損壞或格式錯誤",
	}

	// 參數錯誤
	ErrInvalidParameter = &GameFlowError{
		Code:    "INVALID_PARAMETER",
		Message: "無效的參數",
	}

	// JP錯誤
	ErrJackpotNotEnabled = &GameFlowError{
		Code:    "JACKPOT_NOT_ENABLED",
		Message: "此遊戲未啟用JP",
	}
)

// NewGameFlowError 創建新的遊戲流程錯誤
func NewGameFlowError(code, message string) *GameFlowError {
	return &GameFlowError{
		Code:    code,
		Message: message,
	}
}

// NewGameFlowErrorWithFormat 使用格式化字串創建新的遊戲流程錯誤
func NewGameFlowErrorWithFormat(code string, format string, args ...interface{}) *GameFlowError {
	return &GameFlowError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}
