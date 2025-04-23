/**
 * WebSocket 共用工具庫
 * 提供 WebSocket 連接管理、心跳機制與基本訊息處理功能
 */

// WebSocket 狀態常量
const WS_STATE = {
    INITIAL: "StateInitial",
    BETTING: "StateBetting",
    DRAWING: "StateDrawing",
    ENDED: "StateEnded"
};

// 心跳設置
const HEARTBEAT_INTERVAL = 10000; // 10秒發送一次心跳

/**
 * WebSocket 連接管理器
 */
class WebSocketManager {
    constructor(options = {}) {
        this.socket = null;
        this.isConnected = false;
        this.heartbeatInterval = null;
        this.connectionTime = null;
        this.lastHeartbeat = null;
        this.serverUrl = '';
        this.autoReconnect = options.autoReconnect !== false;
        this.reconnectDelay = options.reconnectDelay || 3000;
        this.onConnectionStatusChange = options.onConnectionStatusChange || (() => {});
        this.onMessage = options.onMessage || (() => {});
        this.onGameStateChange = options.onGameStateChange || (() => {});
        this.onLog = options.onLog || (() => {});
    }

    /**
     * 連接到 WebSocket 服務器
     * @param {string} serverUrl - WebSocket 服務器 URL
     * @param {Object} authParams - 認證參數 (可選)
     */
    connect(serverUrl, authParams = {}) {
        if (this.isConnected) return;
        
        this.serverUrl = serverUrl.trim();
        if (!this.serverUrl) {
            this.log('錯誤：請輸入服務器地址', 'error');
            return;
        }
        
        // 添加認證參數到 URL (如果提供)
        let fullUrl = this.serverUrl;
        if (authParams && Object.keys(authParams).length > 0) {
            const urlObj = new URL(this.serverUrl);
            Object.entries(authParams).forEach(([key, value]) => {
                if (value) urlObj.searchParams.append(key, value);
            });
            fullUrl = urlObj.toString();
        }
        
        try {
            this.socket = new WebSocket(fullUrl);
            
            this.socket.onopen = (e) => {
                this.isConnected = true;
                this.connectionTime = new Date();
                this.updateConnectionStatus(true);
                this.startHeartbeat();
                this.log('已成功連接到服務器', 'system');
            };
            
            this.socket.onmessage = (event) => {
                this.handleMessage(event.data);
            };
            
            this.socket.onclose = (event) => {
                this.stopHeartbeat();
                if (event.wasClean) {
                    this.log(`連接已關閉，代碼=${event.code} 原因=${event.reason}`, 'system');
                } else {
                    this.log('連接意外中斷', 'error');
                    // 如果是意外斷線且不是手動斷開，嘗試重新連接
                    if (this.isConnected && this.autoReconnect) {
                        this.log('自動嘗試重新連接...', 'system');
                        setTimeout(() => this.connect(this.serverUrl, authParams), this.reconnectDelay);
                    }
                }
                this.updateConnectionStatus(false);
            };
            
            this.socket.onerror = (error) => {
                this.log(`WebSocket 錯誤：${error.message || '連接錯誤'}`, 'error');
            };
        } catch (error) {
            this.log(`連接錯誤：${error.message}`, 'error');
        }
    }

    /**
     * 斷開 WebSocket 連接
     */
    disconnect() {
        if (!this.isConnected || !this.socket) return;
        
        this.stopHeartbeat();
        this.socket.close();
        this.socket = null;
    }

    /**
     * 發送訊息到服務器
     * @param {Object} message - 要發送的訊息物件
     */
    sendMessage(message) {
        if (!this.isConnected || !this.socket || this.socket.readyState !== WebSocket.OPEN) {
            this.log('無法發送訊息：未連接', 'error');
            return false;
        }
        
        try {
            const messageStr = typeof message === 'string' ? message : JSON.stringify(message);
            this.socket.send(messageStr);
            this.log(`發送：${messageStr}`, 'sent');
            return true;
        } catch (error) {
            this.log(`發送訊息錯誤：${error.message}`, 'error');
            return false;
        }
    }

    /**
     * 啟動心跳機制
     */
    startHeartbeat() {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
        }
        
        this.heartbeatInterval = setInterval(() => {
            if (this.socket && this.socket.readyState === WebSocket.OPEN) {
                // 發送心跳訊息
                const heartbeatMsg = {
                    type: "heartbeat",
                    timestamp: Date.now()
                };
                this.socket.send(JSON.stringify(heartbeatMsg));
                this.lastHeartbeat = new Date();
                this.updateConnectionInfo();
                this.log('發送心跳...', 'system', true); // 心跳日誌可選靜默
            } else {
                this.log('心跳失敗: 連接已關閉', 'error');
                this.stopHeartbeat();
            }
        }, HEARTBEAT_INTERVAL);
    }

    /**
     * 停止心跳機制
     */
    stopHeartbeat() {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
    }

    /**
     * 更新連接資訊
     */
    updateConnectionInfo() {
        if (!this.isConnected) {
            return {
                heartbeat: '--',
                connTime: '--',
                connected: false
            };
        }
        
        let connTimeStr = '剛剛連接';
        let heartbeatStr = '尚未發送';
        
        if (this.connectionTime) {
            const connDuration = Math.floor((new Date() - this.connectionTime) / 1000);
            if (connDuration < 60) {
                connTimeStr = `${connDuration} 秒`;
            } else if (connDuration < 3600) {
                connTimeStr = `${Math.floor(connDuration / 60)} 分鐘`;
            } else {
                connTimeStr = `${Math.floor(connDuration / 3600)} 小時 ${Math.floor((connDuration % 3600) / 60)} 分鐘`;
            }
        }
        
        if (this.lastHeartbeat) {
            const heartbeatTime = this.lastHeartbeat.toLocaleTimeString();
            heartbeatStr = heartbeatTime;
        }
        
        const info = {
            heartbeat: heartbeatStr,
            connTime: connTimeStr,
            connected: true
        };
        
        // 觸發回調
        if (typeof this.onConnectionStatusChange === 'function') {
            this.onConnectionStatusChange(info);
        }
        
        return info;
    }

    /**
     * 更新連接狀態
     * @param {boolean} connected - 連接狀態
     */
    updateConnectionStatus(connected) {
        this.isConnected = connected;
        
        if (!connected) {
            this.connectionTime = null;
            this.lastHeartbeat = null;
        }
        
        this.updateConnectionInfo();
    }

    /**
     * 處理接收到的訊息
     * @param {string} data - 接收到的數據
     */
    handleMessage(data) {
        this.log(`收到：${data}`, 'received');
        
        try {
            const message = JSON.parse(data);
            
            // 處理心跳響應
            if (message.type === 'heartbeat') {
                this.lastHeartbeat = new Date();
                this.updateConnectionInfo();
                return;
            }
            
            // 檢查是否為命令訊息
            if (message.command) {
                this.handleCommand(message);
            }
            
            // 觸發消息回調
            if (typeof this.onMessage === 'function') {
                this.onMessage(message);
            }
        } catch (error) {
            this.log(`解析訊息錯誤：${error.message}`, 'error');
        }
    }

    /**
     * 處理命令訊息
     * @param {Object} message - 命令訊息物件
     */
    handleCommand(message) {
        const command = message.command;
        const state = message.state;
        
        // 更新遊戲狀態顯示
        if (state && typeof this.onGameStateChange === 'function') {
            this.onGameStateChange(state, message);
        }
    }

    /**
     * 記錄訊息
     * @param {string} message - 要記錄的訊息
     * @param {string} type - 訊息類型 (system, sent, received, error)
     * @param {boolean} silent - 是否靜默記錄 (不觸發回調)
     */
    log(message, type = 'system', silent = false) {
        if (!silent && typeof this.onLog === 'function') {
            const timestamp = new Date().toLocaleTimeString();
            this.onLog(message, type, timestamp);
        }
    }

    /**
     * 獲取格式化的遊戲狀態文本
     * @param {string} state - 遊戲狀態代碼
     * @returns {string} 格式化的狀態文本
     */
    static getGameStateText(state) {
        let stateText = '未知';
        
        switch (state) {
            case WS_STATE.INITIAL:
                stateText = '初始化';
                break;
            case WS_STATE.BETTING:
                stateText = '下注中';
                break;
            case WS_STATE.DRAWING:
                stateText = '開獎中';
                break;
            case WS_STATE.ENDED:
                stateText = '已結束';
                break;
            default:
                stateText = state;
        }
        
        return stateText;
    }
}

// 導出工具庫
window.WebSocketManager = WebSocketManager;
window.WS_STATE = WS_STATE; 