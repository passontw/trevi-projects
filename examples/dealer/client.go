package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// 命令列參數
var (
	addr     = flag.String("addr", "localhost:8080", "WebSocket 伺服器位址")
	endpoint = flag.String("endpoint", "/ws", "WebSocket 端點")
)

func main() {
	flag.Parse()

	// 設定中斷訊號處理
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// 建立 WebSocket URL
	u := url.URL{Scheme: "ws", Host: *addr, Path: *endpoint}
	log.Printf("正在連線至 %s", u.String())

	// 連線至 WebSocket 伺服器
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("無法連線: %v", err)
	}
	defer c.Close()

	// 建立取消上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 讀取訊息的協程
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Printf("讀取錯誤: %v", err)
				return
			}
			log.Printf("收到訊息: %s", message)
		}
	}()

	// 監聽用戶輸入並發送訊息
	scanner := bufio.NewScanner(os.Stdin)
	log.Println("請輸入訊息，按 Enter 發送，輸入 'exit' 離開:")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if scanner.Scan() {
					text := scanner.Text()
					if strings.ToLower(text) == "exit" {
						cancel()
						return
					}

					// 發送訊息至伺服器
					err := c.WriteMessage(websocket.TextMessage, []byte(text))
					if err != nil {
						log.Printf("發送錯誤: %v", err)
						cancel()
						return
					}
				} else {
					if err := scanner.Err(); err != nil {
						log.Printf("輸入錯誤: %v", err)
					}
					cancel()
					return
				}
			}
		}
	}()

	// 保持連線並發送 ping
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			log.Println("正在關閉連線...")

			// 發送關閉訊息
			err := c.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("無法發送關閉訊息: %v", err)
			}

			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		case <-ticker.C:
			// 發送 ping 訊息
			err := c.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				log.Printf("Ping 錯誤: %v", err)
				return
			}
		case <-interrupt:
			log.Println("收到中斷信號，正在關閉連線...")

			// 發送關閉訊息
			err := c.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Printf("無法發送關閉訊息: %v", err)
			}

			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
