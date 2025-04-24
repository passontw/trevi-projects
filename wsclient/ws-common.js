/**
 * WebSocket共用管理工具
 * 提供WebSocket連線管理、心跳機制和基本訊息處理
 * 
 * 版本: 1.1.0
 * 更新: 增強錯誤處理、改進連接狀態管理、添加詳細日誌
 */
class WebSocketManager {
    /**
     * 初始化WebSocket管理器
     * @param {Object} options 選項配置
     * @param {Function} options.onConnectionStatusChange 連接狀態變更處理函數
     * @param {Function} options.onMessage 消息處理函數
     * @param {Function} options.onGameStateChange 遊戲狀態變更處理函數
     * @param {Function} options.onLog 日誌處理函數
     * @param {Function} options.onError 錯誤處理函數
     * @param {Function} options.onReconnect 重連嘗試時的回調函數
     * @param {number} options.heartbeatInterval 心跳間隔(毫秒)，預設10秒
     * @param {boolean} options.autoReconnect 是否自動重連，預設true
     * @param {number} options.reconnectDelay 重連延遲(毫秒)，預設3秒
     * @param {number} options.connectionTimeout 連接超時(毫秒)，預設10秒
     * @param {number} options.maxReconnectAttempts 最大重連次數，預設5次
     * @param {boolean} options.debug 是否啟用調試模式，預設false
     */
    constructor(options = {}) {
        this.options = {
            heartbeatInterval: options.heartbeatInterval || 10000,
            autoReconnect: options.autoReconnect !== false,
            reconnectDelay: options.reconnectDelay || 3000,
            connectionTimeout: options.connectionTimeout || 10000,
            maxReconnectAttempts: options.maxReconnectAttempts || 5,
            debug: options.debug || false
        };
        
        // 回調函數
        this.onConnectionStatusChange = options.onConnectionStatusChange;
        this.onMessage = options.onMessage;
        this.onGameStateChange = options.onGameStateChange;
        this.onLog = options.onLog;
        this.onError = options.onError;
        this.onReconnect = options.onReconnect;
        
        // WebSocket狀態
        this.ws = null;
        this.isConnected = false;
        this.serverUrl = '';
        this.authParams = {};
        this.reconnectAttempts = 0;
        
        // 定時器
        this.heartbeatTimer = null;
        this.connectionTimeoutTimer = null;
        this.reconnectTimer = null;
        
        // 連接信息
        this.connectionTime = null;
        this.lastHeartbeat = null;
        this.lastPingSent = null;
        this.pingPongTime = null;
        
        // 連接狀態歷史記錄
        this.connectionHistory = [];
        
        // 遊戲狀態
        this.currentGameState = null;

        // 統計數據
        this.stats = {
            messagesReceived: 0,
            messagesSent: 0,
            reconnectAttempts: 0,
            totalConnections: 0,
            errors: 0
        };
        
        // 聲音效果相關方法
        this.sounds = {};
        this.enableSounds = true;
        
        this.debugLog('WebSocketManager初始化完成', options);
    }
    
    /**
     * 連接到WebSocket服務器
     * @param {string} url WebSocket服務器URL
     * @param {Object} authParams 認證參數
     * @returns {Promise} 連接結果的Promise
     */
    connect(url, authParams = {}) {
        return new Promise((resolve, reject) => {
            if (this.isConnected && this.ws) {
                const error = new Error('已經連接到服務器，請先斷開連接');
                this.log(error.message, 'error');
                reject(error);
                return;
            }
            
            // 清除所有舊定時器
            this._clearAllTimers();
            
            // 重置重連嘗試次數
            this.reconnectAttempts = 0;
            
            // 保存連接信息
            this.serverUrl = url;
            this.authParams = authParams || {};
            
            // 添加連接開始記錄
            this._addConnectionHistoryEntry('connect_attempt', {
                url: this.serverUrl
            });
            
            // 嘗試建立連接
            this._establishConnection(resolve, reject);
        });
    }
    
    /**
     * 建立WebSocket連接
     * @param {Function} resolve Promise resolve函數
     * @param {Function} reject Promise reject函數
     * @private
     */
    _establishConnection(resolve, reject) {
        try {
            this.log(`正在連接到 ${this.serverUrl}...`, 'system');
            
            // 創建WebSocket實例
            this.ws = new WebSocket(this.serverUrl);
            
            // 設置連接超時
            this._setConnectionTimeout(reject);
            
            // 設置事件處理器
            this._setupEventHandlers(resolve, reject);
            
        } catch (error) {
            this.log(`連接失敗: ${error.message}`, 'error');
            this._handleError(error);
            
            // 添加連接失敗記錄
            this._addConnectionHistoryEntry('connect_error', {
                error: error.message
            });
            
            // 如果配置了自動重連，進行重連
            if (this.options.autoReconnect) {
                this._scheduleReconnect();
            }
            
            // 拒絕Promise
            if (reject) reject(error);
        }
    }
    
    /**
     * 設置連接超時定時器
     * @param {Function} reject Promise reject函數
     * @private
     */
    _setConnectionTimeout(reject) {
        // 清除現有的超時定時器
        if (this.connectionTimeoutTimer) {
            clearTimeout(this.connectionTimeoutTimer);
        }
        
        // 設置新的超時定時器
        this.connectionTimeoutTimer = setTimeout(() => {
            if (this.ws && this.ws.readyState !== WebSocket.OPEN) {
                const error = new Error(`連接超時 (${this.options.connectionTimeout}ms)`);
                this.log(error.message, 'error');
                this._handleError(error);
                
                // 添加連接超時記錄
                this._addConnectionHistoryEntry('connect_timeout', {
                    timeout: this.options.connectionTimeout
                });
                
                // 清理資源
                if (this.ws) {
                    this.ws.close();
                    this.ws = null;
                }
                
                // 如果配置了自動重連，進行重連
                if (this.options.autoReconnect) {
                    this._scheduleReconnect();
                }
                
                // 拒絕Promise
                if (reject) reject(error);
            }
        }, this.options.connectionTimeout);
    }
    
    /**
     * 設置WebSocket事件處理器
     * @param {Function} resolve Promise resolve函數
     * @param {Function} reject Promise reject函數
     * @private
     */
    _setupEventHandlers(resolve, reject) {
        if (!this.ws) return;
        
        // 連接建立事件
        this.ws.onopen = () => {
            // 清除連接超時定時器
            if (this.connectionTimeoutTimer) {
                clearTimeout(this.connectionTimeoutTimer);
                this.connectionTimeoutTimer = null;
            }
            
            this.isConnected = true;
            this.connectionTime = new Date();
            this.reconnectAttempts = 0;
            this.stats.totalConnections++;
            
            // 添加連接成功記錄
            this._addConnectionHistoryEntry('connected', {
                time: this.connectionTime
            });
            
            this.log('WebSocket連接已建立', 'system');
            
            // 啟動心跳機制
            this._startHeartbeat();
            
            // 發送認證信息
            if (Object.keys(this.authParams).length > 0) {
                this._sendAuthMessage();
            }
            
            // 通知連接狀態變更
            this._notifyConnectionStatusChange();
            
            // 解析Promise
            if (resolve) resolve();
        };
        
        // 接收消息事件
        this.ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.stats.messagesReceived++;
                
                // 處理心跳響應
                if (data.type === 'heartbeat' || data.command === 'heartbeat') {
                    this.lastHeartbeat = new Date();
                    
                    // 計算往返時間
                    if (this.lastPingSent) {
                        this.pingPongTime = this.lastHeartbeat - this.lastPingSent;
                        this.debugLog(`心跳往返時間: ${this.pingPongTime}ms`);
                    }
                    
                    this._notifyConnectionStatusChange();
                    return;
                }
                
                // 處理遊戲狀態更新
                if (data.command === 'gameState' || data.type === 'gameState') {
                    const oldState = this.currentGameState;
                    this.currentGameState = data.state;
                    
                    // 添加遊戲狀態變更記錄
                    this._addConnectionHistoryEntry('game_state_change', {
                        from: oldState,
                        to: this.currentGameState
                    });
                    
                    // 播放遊戲狀態變更音效
                    this.playSound('notification', 0.7);
                    
                    if (this.onGameStateChange) {
                        this.onGameStateChange(data.state, data);
                    }
                }
                
                // 處理抽出的球號
                if (data.command === 'ballDrawn' && data.number !== undefined) {
                    // 播放抽球音效
                    this.playSound('message', 0.5);
                    
                    if (typeof window.addDrawnBall === 'function') {
                        window.addDrawnBall(data.number);
                    }
                }
                
                // 處理錯誤消息
                if (data.error || data.status === 'error') {
                    this.log(`服務器錯誤: ${data.message || data.error || '未知錯誤'}`, 'error');
                    // 播放錯誤音效
                    this.playSound('notification', 1.0);
                    this._handleError(new Error(data.message || data.error || '服務器返回錯誤'));
                }
                
                // 記錄收到的消息
                this.log(`收到: ${JSON.stringify(data)}`, 'received', data.type === 'heartbeat');
                
                // 根據消息類型播放不同音效
                if (!data.type && !data.command && !data.error && data.type !== 'heartbeat') {
                    this.playSound('message', 0.4);
                }
                
                // 調用消息處理回調
                if (this.onMessage) {
                    this.onMessage(data);
                }
            } catch (error) {
                this.log(`解析消息失敗: ${error.message}`, 'error');
                this.log(`原始消息: ${event.data}`, 'error');
                this._handleError(error);
            }
        };
        
        // 連接關閉事件
        this.ws.onclose = (event) => {
            // 清除所有定時器
            this._clearAllTimers();
            
            const wasConnected = this.isConnected;
            this.isConnected = false;
            
            // 添加連接關閉記錄
            this._addConnectionHistoryEntry('disconnected', {
                code: event.code,
                reason: event.reason,
                wasClean: event.wasClean
            });
            
            // 關閉代碼不為1000(正常關閉)且配置了自動重連，嘗試重連
            if (event.code !== 1000 && this.options.autoReconnect) {
                const reason = event.reason ? ` (${event.reason})` : '';
                this.log(`連接意外關閉，代碼: ${event.code}${reason}，嘗試重連...`, 'system');
                this._scheduleReconnect();
            } else {
                let closeMsg = '連接已關閉';
                if (event.code !== 1000) {
                    closeMsg += `，代碼: ${event.code}`;
                    if (event.reason) {
                        closeMsg += ` (${event.reason})`;
                    }
                }
                this.log(closeMsg, 'system');
            }
            
            // 只有之前是連接狀態時才通知狀態變更
            if (wasConnected) {
                this._notifyConnectionStatusChange();
            }
            
            // 如果是異常關閉並且提供了reject函數，則拒絕Promise
            if (event.code !== 1000 && reject) {
                reject(new Error(`WebSocket連接關閉，代碼: ${event.code}, 原因: ${event.reason || '未知'}`));
            }
        };
        
        // 連接錯誤事件
        this.ws.onerror = (error) => {
            const errorMsg = error.message || '未知WebSocket錯誤';
            this.log(`WebSocket錯誤: ${errorMsg}`, 'error');
            
            // 添加錯誤記錄
            this._addConnectionHistoryEntry('error', {
                message: errorMsg
            });
            
            this._handleError(error);
            
            // 如果提供了reject函數，則拒絕Promise
            if (reject) {
                reject(error);
            }
        };
    }
    
    /**
     * 處理錯誤
     * @param {Error} error 錯誤對象
     * @private
     */
    _handleError(error) {
        this.stats.errors++;
        
        if (this.onError) {
            this.onError(error);
        }
    }
    
    /**
     * 發送認證消息
     * @private
     */
    _sendAuthMessage() {
        const authMessage = {
            type: 'auth',
            ...this.authParams
        };
        
        this.sendMessage(authMessage);
        
        // 添加認證嘗試記錄
        this._addConnectionHistoryEntry('auth_attempt', {
            params: {...this.authParams, password: '******'} // 隱藏敏感信息
        });
    }
    
    /**
     * 安排重連
     * @private
     */
    _scheduleReconnect() {
        // 清除現有的重連定時器
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
        }
        
        // 超過最大重連次數
        if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
            this.log(`已達到最大重連次數(${this.options.maxReconnectAttempts})，停止重連`, 'error');
            
            // 添加放棄重連記錄
            this._addConnectionHistoryEntry('reconnect_abandoned', {
                attempts: this.reconnectAttempts,
                maxAttempts: this.options.maxReconnectAttempts
            });
            
            return;
        }
        
        this.reconnectAttempts++;
        this.stats.reconnectAttempts++;
        
        // 計算重連延遲（可以實現指數退避）
        const delay = Math.min(
            this.options.reconnectDelay * Math.pow(1.5, this.reconnectAttempts - 1),
            30000  // 最大30秒的退避
        );
        
        this.log(`將在 ${(delay / 1000).toFixed(1)} 秒後重連 (${this.reconnectAttempts}/${this.options.maxReconnectAttempts})`, 'system');
        
        // 添加重連計劃記錄
        this._addConnectionHistoryEntry('reconnect_scheduled', {
            attempt: this.reconnectAttempts,
            delay: delay
        });
        
        // 調用重連回調
        if (this.onReconnect) {
            this.onReconnect(this.reconnectAttempts, this.options.maxReconnectAttempts, delay);
        }
        
        this.reconnectTimer = setTimeout(() => {
            if (!this.isConnected) {
                // 添加重連嘗試記錄
                this._addConnectionHistoryEntry('reconnect_attempt', {
                    attempt: this.reconnectAttempts,
                    url: this.serverUrl
                });
                
                this._establishConnection();
            }
        }, delay);
    }
    
    /**
     * 啟動心跳機制
     * @private
     */
    _startHeartbeat() {
        // 先停止現有的心跳
        this._stopHeartbeat();
        
        // 設置心跳定時器
        this.heartbeatTimer = setInterval(() => {
            if (this.isConnected && this.ws && this.ws.readyState === WebSocket.OPEN) {
                // 發送心跳消息
                const heartbeatMessage = {
                    type: 'heartbeat',
                    timestamp: Date.now()
                };
                
                this.lastPingSent = new Date();
                this.sendMessage(heartbeatMessage, true);
                
                // 檢查心跳超時
                if (this.lastHeartbeat) {
                    const now = new Date();
                    const heartbeatTimeout = this.options.heartbeatInterval * 3;
                    
                    if (now - this.lastHeartbeat > heartbeatTimeout) {
                        this.log(`心跳超時，上次心跳: ${this.lastHeartbeat.toLocaleTimeString()}`, 'error');
                        
                        // 添加心跳超時記錄
                        this._addConnectionHistoryEntry('heartbeat_timeout', {
                            lastHeartbeat: this.lastHeartbeat,
                            timeout: heartbeatTimeout
                        });
                        
                        this.disconnect();
                        
                        // 如果配置了自動重連，進行重連
                        if (this.options.autoReconnect) {
                            this._scheduleReconnect();
                        }
                    }
                }
            }
        }, this.options.heartbeatInterval);
    }
    
    /**
     * 停止心跳機制
     * @private
     */
    _stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    }
    
    /**
     * 清除所有定時器
     * @private
     */
    _clearAllTimers() {
        // 停止心跳
        this._stopHeartbeat();
        
        // 清除連接超時定時器
        if (this.connectionTimeoutTimer) {
            clearTimeout(this.connectionTimeoutTimer);
            this.connectionTimeoutTimer = null;
        }
        
        // 清除重連定時器
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
    }
    
    /**
     * 通知連接狀態變更
     * @private
     */
    _notifyConnectionStatusChange() {
        const statusInfo = {
            isConnected: this.isConnected,
            connectionTime: this.connectionTime,
            connectionDuration: this.isConnected ? (new Date() - this.connectionTime) : 0,
            connectionTimeFormatted: this._getConnectionTimeFormatted(),
            lastHeartbeat: this.lastHeartbeat,
            pingPongTime: this.pingPongTime,
            reconnectAttempts: this.reconnectAttempts,
            gameState: this.currentGameState,
            gameStateText: WebSocketManager.getGameStateText(this.currentGameState)
        };
        
        // 音效通知連接狀態變更
        if (this.isConnected && this.lastStatusInfo && !this.lastStatusInfo.isConnected) {
            // 連接成功音效
            this.playSound('notification', 0.6);
        } else if (!this.isConnected && this.lastStatusInfo && this.lastStatusInfo.isConnected) {
            // 連接斷開音效
            this.playSound('notification', 0.8);
        }
        
        // 儲存上一次狀態信息
        this.lastStatusInfo = {...statusInfo};
        
        // 調用狀態變更回調
        if (this.onConnectionStatusChange) {
            this.onConnectionStatusChange(statusInfo);
        }
        
        return statusInfo;
    }
    
    /**
     * 獲取格式化的連接時間
     * @returns {string} 格式化的連接時間
     * @private
     */
    _getConnectionTimeFormatted() {
        if (!this.isConnected || !this.connectionTime) {
            return '未連接';
        }
        
        const now = new Date();
        const diffMs = now - this.connectionTime;
        const seconds = Math.floor(diffMs / 1000);
        
        if (seconds < 60) {
            return `${seconds} 秒`;
        } else if (seconds < 3600) {
            const minutes = Math.floor(seconds / 60);
            const remainingSeconds = seconds % 60;
            return `${minutes} 分 ${remainingSeconds} 秒`;
        } else {
            const hours = Math.floor(seconds / 3600);
            const minutes = Math.floor((seconds % 3600) / 60);
            return `${hours} 小時 ${minutes} 分`;
        }
    }
    
    /**
     * 添加連接歷史記錄
     * @param {string} type 記錄類型
     * @param {Object} data 記錄數據
     * @private
     */
    _addConnectionHistoryEntry(type, data = {}) {
        // 限制歷史記錄最大長度為100條
        if (this.connectionHistory.length >= 100) {
            this.connectionHistory.shift();
        }
        
        this.connectionHistory.push({
            type,
            time: new Date(),
            data
        });
        
        this.debugLog(`連接事件: ${type}`, data);
    }
    
    /**
     * 發送消息
     * @param {Object} data 要發送的數據
     * @param {boolean} skipLog 是否跳過日誌記錄
     * @returns {boolean} 是否成功發送
     */
    sendMessage(data, skipLog = false) {
        if (!this.isConnected || !this.ws || this.ws.readyState !== WebSocket.OPEN) {
            this.log('無法發送消息：WebSocket未連接', 'error');
            return false;
        }
        
        try {
            const message = JSON.stringify(data);
            this.ws.send(message);
            this.stats.messagesSent++;
            
            if (!skipLog && data.type !== 'heartbeat') {
                this.log(`發送: ${JSON.stringify(data)}`, 'sent', true);
            }
            
            return true;
        } catch (error) {
            this.log(`發送消息失敗: ${error.message}`, 'error');
            this._handleError(error);
            return false;
        }
    }
    
    /**
     * 斷開連接
     * @param {number} code 關閉代碼
     * @param {string} reason 關閉原因
     * @returns {Promise} 表示斷開連接操作的Promise
     */
    disconnect(code = 1000, reason = '用戶主動斷開連接') {
        return new Promise((resolve) => {
            // 清除所有定時器
            this._clearAllTimers();
            
            // 關閉WebSocket連接
            if (this.ws) {
                // 添加主動斷開記錄
                this._addConnectionHistoryEntry('disconnect_initiated', {
                    code,
                    reason
                });
                
                // 使用正常關閉代碼
                try {
                    this.ws.close(code, reason);
                    this.log(`正在斷開連接: ${reason}`, 'system');
                } catch (e) {
                    this.log(`關閉連接時發生錯誤: ${e.message}`, 'error');
                    this._handleError(e);
                }
                this.ws = null;
            }
            
            this.isConnected = false;
            this._notifyConnectionStatusChange();
            
            // 確保事件循環已完成
            setTimeout(() => {
                resolve();
            }, 0);
        });
    }
    
    /**
     * 記錄日誌
     * @param {string} message 日誌消息
     * @param {string} type 日誌類型
     * @param {boolean} skipDisplay 是否跳過顯示
     */
    log(message, type = 'system', skipDisplay = false) {
        const now = new Date();
        const timestamp = now.toLocaleTimeString('zh-TW', {
            hour12: false,
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
            fractionalSecondDigits: 3
        });
        
        const fullMessage = `[${timestamp}] ${message}`;
        
        if (this.onLog) {
            this.onLog(fullMessage, type, skipDisplay);
        } else if (!skipDisplay) {
            const typeText = type === 'error' ? '[錯誤]' : 
                             type === 'received' ? '[接收]' : 
                             type === 'sent' ? '[發送]' : '[系統]';
            
            console.log(`${typeText} ${fullMessage}`);
        }
    }
    
    /**
     * 調試日誌
     * @param {string} message 日誌消息
     * @param {any} data 調試數據
     * @private
     */
    debugLog(message, data = null) {
        if (this.options.debug) {
            if (data) {
                console.debug(`[WebSocketManager調試] ${message}`, data);
            } else {
                console.debug(`[WebSocketManager調試] ${message}`);
            }
        }
    }
    
    /**
     * 檢查是否已連接
     * @returns {boolean} 是否已連接
     */
    isConnectionActive() {
        return this.isConnected && this.ws && this.ws.readyState === WebSocket.OPEN;
    }
    
    /**
     * 獲取連接信息
     * @returns {Object} 連接信息
     */
    getConnectionInfo() {
        return {
            connected: this.isConnected,
            serverUrl: this.serverUrl,
            connectionTime: this.connectionTime,
            lastHeartbeat: this.lastHeartbeat,
            reconnectAttempts: this.reconnectAttempts,
            formattedConnTime: this._getConnectionTimeFormatted(),
            pingPongTime: this.pingPongTime,
            stats: this.stats,
            connectionHistoryCount: this.connectionHistory.length
        };
    }
    
    /**
     * 獲取連接歷史記錄
     * @param {number} limit 最大記錄數
     * @returns {Array} 連接歷史記錄
     */
    getConnectionHistory(limit = 20) {
        return this.connectionHistory.slice(-limit);
    }
    
    /**
     * 重置統計數據
     */
    resetStats() {
        this.stats = {
            messagesReceived: 0,
            messagesSent: 0,
            reconnectAttempts: 0,
            totalConnections: this.stats.totalConnections,
            errors: 0
        };
        
        this.log('統計數據已重置', 'system');
    }
    
    /**
     * 獲取遊戲狀態文本
     * @param {string} state 遊戲狀態
     * @returns {string} 遊戲狀態文本
     * @static
     */
    static getGameStateText(state) {
        switch (state) {
            case WS_STATE.INITIAL: return '初始化';
            case WS_STATE.BETTING: return '下注中';
            case WS_STATE.DRAWING: return '開獎中';
            case WS_STATE.ENDED: return '已結束';
            default: return state || '未知';
        }
    }
    
    // 聲音效果相關方法
    
    /**
     * 加載自訂音效
     * @param {Object} customSounds - 自訂音效對象，格式：{名稱: 音效文件路徑}
     */
    loadSounds(customSounds = {}) {
        try {
            for (const [name, path] of Object.entries(customSounds)) {
                this.sounds[name] = new Audio(path);
                this.sounds[name].load();
                this._log(`音效 "${name}" 已加載: ${path}`);
            }
        } catch (error) {
            this._log(`加載音效時發生錯誤: ${error.message}`, 'error');
        }
    }
    
    /**
     * 播放指定音效
     * @param {string} soundName - 要播放的音效名稱
     * @param {number} volume - 播放音量 (0-1)
     * @returns {boolean} - 是否成功播放
     */
    playSound(soundName, volume = 1.0) {
        if (!this.enableSounds) return false;
        
        try {
            const sound = this.sounds[soundName];
            if (!sound) {
                this._log(`音效 "${soundName}" 不存在`, 'warn');
                return false;
            }
            
            // 暫停並重置當前播放
            sound.pause();
            sound.currentTime = 0;
            
            // 設置音量並播放
            sound.volume = Math.max(0, Math.min(1, volume));
            sound.play().catch(err => {
                this._log(`播放音效 "${soundName}" 時發生錯誤: ${err.message}`, 'error');
            });
            
            return true;
        } catch (error) {
            this._log(`播放音效時發生錯誤: ${error.message}`, 'error');
            return false;
        }
    }
    
    /**
     * 啟用或禁用所有音效
     * @param {boolean} enabled - 是否啟用音效
     */
    toggleSounds(enabled) {
        this.enableSounds = enabled;
        this._log(`音效已${enabled ? '啟用' : '禁用'}`);
    }
    
    /**
     * 停止所有正在播放的音效
     */
    stopAllSounds() {
        try {
            Object.values(this.sounds).forEach(sound => {
                sound.pause();
                sound.currentTime = 0;
            });
        } catch (error) {
            this._log(`停止音效時發生錯誤: ${error.message}`, 'error');
        }
    }
    
    /**
     * 預加載所有音效以備用
     */
    preloadAllSounds() {
        try {
            Object.values(this.sounds).forEach(sound => {
                sound.load();
            });
            this._log('所有音效已預加載');
        } catch (error) {
            this._log(`預加載音效時發生錯誤: ${error.message}`, 'error');
        }
    }
}

// 遊戲狀態常量
const WS_STATE = {
    INITIAL: 'initial',
    BETTING: 'betting',
    DRAWING: 'drawing',
    ENDED: 'ended'
};

// 導出工具庫
window.WebSocketManager = WebSocketManager;