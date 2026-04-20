package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"ssh-port-forwarder/internal/service"
	"ssh-port-forwarder/internal/service/health"
)

type WSHandler struct {
	container *service.Container
}

func NewWSHandler(c *service.Container) *WSHandler {
	return &WSHandler{container: c}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境可配置
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Status WebSocket 状态推送端点
func (h *WSHandler) Status(c *gin.Context) {
	// 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// 设置读取超时和 pong 处理
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 订阅 HealthChecker 的 WebSocket 事件通道
	eventCh := h.container.HealthChecker.SubscribeWS()
	defer h.container.HealthChecker.UnsubscribeWS(eventCh)

	// 启动 ping 定时器
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 启动 goroutine 处理客户端消息（ping/pong）
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[WS] Read error: %v", err)
				}
				return
			}
		}
	}()

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				// 通道关闭，发送关闭消息并退出
				h.sendCloseMessage(conn)
				return
			}
			if err := h.sendEvent(conn, event); err != nil {
				log.Printf("[WS] Failed to send event: %v", err)
				return
			}

		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[WS] Failed to send ping: %v", err)
				return
			}

		case <-done:
			return
		}
	}
}

func (h *WSHandler) sendEvent(conn *websocket.Conn, event health.HealthEvent) error {
	msg := WSMessage{
		Type: "host_status_change",
		Data: map[string]interface{}{
			"host_id":       event.HostID,
			"health_status": event.HealthStatus,
			"health_score":  event.HealthScore,
			"latency_ms":    event.LatencyMs,
			"checked_at":    event.CheckedAt,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

func (h *WSHandler) sendCloseMessage(conn *websocket.Conn) {
	msg := WSMessage{
		Type: "connection_closed",
		Data: map[string]string{
			"reason": "server_shutdown",
		},
	}

	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.CloseMessage, data)
}
