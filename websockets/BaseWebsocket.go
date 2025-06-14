package websockets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	ws "github.com/gorilla/websocket"
)

const (
	HEARTBEAT_CHECK_INTERVAL_SEC        = 5
	HEARTBEAT_CLOSE_ON_NO_HEARTBEAT_SEC = 20
)

// type websocket_CombinedStream_Message struct {
// 	Stream string              `json:"stream"`
// 	Data   jsoniter.RawMessage `json:"data"`
// }

// This websocket handles simply connecting to binance, reading the messages and successfully disconnecting once the connection goes dead
//
// And handles sending requests via the Id system
type baseWebsocket struct {
	// Host server's URL
	url                     string
	lastHeartbeat_Timestamp time.Time
	closed                  atomic.Bool

	conn    *ws.Conn
	writeMu sync.Mutex

	OnMessage func(messageType int, msg []byte)
	OnError   func(err error)
	OnClose   func(code int, reason string)
}

func (socket *baseWebsocket) init(conn *ws.Conn, URL string) {
	socket.url = URL
	socket.lastHeartbeat_Timestamp = time.Now()
	socket.conn = conn

	////

	socket.conn.SetPingHandler(socket.onPing)
	socket.conn.SetPongHandler(socket.onPong)
	socket.conn.SetCloseHandler(socket.closeHandler)

	////

	go socket.listen()
	go socket.checkHeartbeats()
}

func (socket *baseWebsocket) markAsClosed(code int, reason string) {
	// CompareAndSwap returns "true" if the swap was successful
	// Meaning that socket.closed was false, which means that if it were true, we'd want to emit the OnClose()
	wasFalseAndHasBeenSwappedToTrue := socket.closed.CompareAndSwap(false, true)

	if !wasFalseAndHasBeenSwappedToTrue {
		Logger.DEBUG(fmt.Sprintf("[%s] socket already marked as closed", socket.url))
		return
	}

	if socket.OnClose != nil {
		socket.OnClose(code, reason)
	}
}

func (socket *baseWebsocket) onPing(pingData string) error {
	Logger.DEBUG(fmt.Sprintf("Received a ping: %s", pingData))

	socket.recordLastHeartbeat() // Logs a heartbeat

	err := socket.sendPong([]byte(pingData))
	if err != nil {
		Logger.ERROR("Error sending Pong:", err)
		socket.onError(err)
		return err
	}

	return nil
}

func (socket *baseWebsocket) onPong(appData string) error {
	socket.recordLastHeartbeat()

	return nil
}

func (socket *baseWebsocket) closeHandler(code int, text string) error {
	Logger.DEBUG(fmt.Sprintf("[*Websocket.CloseHandler()] code: %d, text: %s, isClosed: %v", code, text, socket.closed.Load()))
	socket.markAsClosed(code, text)

	return nil
}

func (socket *baseWebsocket) onMessage(messageType int, msg []byte) {
	if socket.closed.Load() {
		return
	}

	if socket.OnMessage != nil {
		socket.OnMessage(messageType, msg)
	}
}

func (socket *baseWebsocket) onError(err error) {
	if socket.OnError != nil {
		socket.OnError(err)
	}

	socket.markAsClosed(ws.CloseInternalServerErr, err.Error())
}

func (socket *baseWebsocket) recordLastHeartbeat() {
	socket.lastHeartbeat_Timestamp = time.Now()
}

func (socket *baseWebsocket) listen() {
	// Goroutine to read messages
	for {
		if socket.closed.Load() {
			return
		}
		msgType, msg, err := socket.conn.ReadMessage()
		if err != nil {
			Logger.ERROR(fmt.Sprintf("[%s] Error reading message", socket.url), err)
			socket.onError(err)
			return
		}

		Logger.DEBUG(fmt.Sprintf("Type: %d, message: %s\n", msgType, string(msg)))
		socket.recordLastHeartbeat()

		socket.onMessage(msgType, msg)
	}
}

func (socket *baseWebsocket) checkHeartbeats() {
	ticker := time.NewTicker(HEARTBEAT_CHECK_INTERVAL_SEC * time.Second)
	defer ticker.Stop()

	for {

		<-ticker.C // Wait the appropriate amount of time
		if socket.closed.Load() {
			Logger.DEBUG("[HEARTBEAT] Websocket is closed, skipping check.")

			return
		}

		currentTime := time.Now()

		elapsed := currentTime.Unix() - socket.lastHeartbeat_Timestamp.Unix()

		// Check if the last heartbeat is older than the close interval
		if elapsed >= HEARTBEAT_CLOSE_ON_NO_HEARTBEAT_SEC {
			Logger.DEBUG("[HEARTBEAT] No heartbeat detected, socket will terminate.")
			return
		}

		// Check if the last heartbeat is older than the heartbeat check interval
		if elapsed >= HEARTBEAT_CHECK_INTERVAL_SEC {
			err := socket.sendPing()
			if err != nil {
				Logger.ERROR("[HEARTBEAT] Error sending ping", err)
				socket.onError(err)
			} else {
				Logger.DEBUG("[HEARTBEAT] Ping sent.")
			}
		}
	}
}

func (socket *baseWebsocket) sendPing() error {
	socket.writeMu.Lock()
	defer socket.writeMu.Unlock()

	// Get current UNIX timestamp in int64 and encode it
	timestamp := time.Now().UnixMilli()
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, timestamp)

	return socket.conn.WriteMessage(ws.PingMessage, buf.Bytes())
}

func (socket *baseWebsocket) sendPong(data []byte) error {
	socket.writeMu.Lock()
	defer socket.writeMu.Unlock()

	return socket.conn.WriteMessage(ws.PongMessage, data)
}

//// Public methods

func (socket *baseWebsocket) SendText(text string) error {
	socket.writeMu.Lock()
	defer socket.writeMu.Unlock()

	return socket.conn.WriteMessage(ws.TextMessage, []byte(text))
}

func (socket *baseWebsocket) SendJSON(v interface{}) error {
	socket.writeMu.Lock()
	defer socket.writeMu.Unlock()

	return socket.conn.WriteJSON(v)
}

func (socket *baseWebsocket) SendPreparedMessage(preparedMessage *ws.PreparedMessage) error {
	socket.writeMu.Lock()
	defer socket.writeMu.Unlock()

	return socket.conn.WritePreparedMessage(preparedMessage)
}

func (socket *baseWebsocket) Close() {
	socket.conn.Close()

	socket.markAsClosed(ws.CloseNormalClosure, "Normal Closure")
}

////

func createBaseSocket(URL string, httpHeader http.Header) (*baseWebsocket, error) {
	conn, _, err := ws.DefaultDialer.Dial(URL, httpHeader)
	if err != nil {
		Logger.ERROR("There was an error creating websocket", err)
		return nil, err
	}
	Logger.DEBUG(fmt.Sprintf("Socket connected: %v", URL))

	var socket baseWebsocket
	socket.init(conn, URL)

	return &socket, nil
}

func assignBaseSocket(conn *ws.Conn, URL string) *baseWebsocket {
	var socket baseWebsocket
	socket.init(conn, URL)

	return &socket
}
