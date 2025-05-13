@echo off
setlocal enabledelayedexpansion

:: 設置顏色
set "GREEN=[92m"
set "YELLOW=[93m"
set "RED=[91m"
set "NC=[0m"

echo %GREEN%===== 樂透自動荷官啟動腳本 =====%NC%

:: 檢查Go環境
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo %RED%錯誤: 未找到Go。請確保Go已安裝且在PATH中。%NC%
    exit /b 1
)

:: 獲取Go版本
for /f "tokens=3" %%i in ('go version') do set GO_VERSION=%%i
set GO_VERSION=%GO_VERSION:go=%

:: 檢查版本號 (簡化版，只檢查是否為1開頭)
echo %YELLOW%檢測到Go版本: %GO_VERSION%%NC%
if not "%GO_VERSION:~0,1%"=="1" (
    echo %RED%警告: 可能不是兼容的Go版本。建議使用Go 1.16或更高版本。%NC%
    timeout /t 3 >nul
)

:: 設置默認參數
if "%SERVER_ADDR%"=="" (
    set "SERVER_ADDR=localhost:8080"
)
if "%ROOM_ID%"=="" (
    set "ROOM_ID=SG01"
)
if "%CONFIG_FILE%"=="" (
    set "CONFIG_FILE=config.json"
)

echo %YELLOW%服務器地址: %SERVER_ADDR%%NC%
echo %YELLOW%房間ID: %ROOM_ID%%NC%
echo %YELLOW%配置文件: %CONFIG_FILE%%NC%

:: 檢查配置文件是否存在
if not exist "%CONFIG_FILE%" (
    echo %YELLOW%警告: 配置文件 '%CONFIG_FILE%' 不存在，將使用預設配置%NC%
)

:: 編譯程序
echo %GREEN%編譯自動荷官程序...%NC%
go build -o autoDealer.exe main.go

if %ERRORLEVEL% NEQ 0 (
    echo %RED%編譯失敗，請檢查錯誤信息。%NC%
    exit /b 1
)

echo %GREEN%編譯成功，正在啟動自動荷官...%NC%

:: 運行程序
set "SERVER_ADDR=%SERVER_ADDR%"
set "ROOM_ID=%ROOM_ID%"
set "CONFIG_FILE=%CONFIG_FILE%"
autoDealer.exe

:: 運行結束後清理
echo %GREEN%自動荷官已退出，清理臨時文件...%NC%
del /f /q autoDealer.exe

echo %GREEN%完成！%NC%
pause 