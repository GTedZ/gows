package websockets

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
)

type pendingRequest struct {
	id   string
	once sync.Once
	ch   chan []byte
}

type privateMessageWebsocket struct {
	base *baseWebsocket

	privateMessagePropertyName string
	isServer                   bool

	pendingRequests struct {
		Mu  sync.Mutex
		Map map[string]*pendingRequest
	}

	OnMessage func(messageType int, msg []byte)
	OnError   func(err error)
	OnClose   func(code int, reason string)
}

func (socket *privateMessageWebsocket) init(baseSocket *baseWebsocket, privateMessagePropertyName string, isServer bool) {
	socket.base = baseSocket

	socket.privateMessagePropertyName = privateMessagePropertyName
	socket.isServer = isServer

	socket.pendingRequests.Map = make(map[string]*pendingRequest)

	//

	socket.base.OnMessage = socket.onMessage
	socket.base.OnError = socket.onError
	socket.base.OnClose = socket.onClose
}

func (socket *privateMessageWebsocket) addPendingRequest() (request *pendingRequest) {
	socket.pendingRequests.Mu.Lock()
	defer socket.pendingRequests.Mu.Unlock()

	var requestId string
	for {
		requestId = generateRandomString()
		_, exists := socket.pendingRequests.Map[requestId]
		if exists {
			continue
		}
		break
	}

	request = &pendingRequest{id: requestId, once: sync.Once{}, ch: make(chan []byte)}
	socket.pendingRequests.Map[requestId] = request

	return request
}

func (socket *privateMessageWebsocket) getPendingRequest(id string) (request *pendingRequest, exists bool) {
	socket.pendingRequests.Mu.Lock()
	defer socket.pendingRequests.Mu.Unlock()

	request, exists = socket.pendingRequests.Map[id]
	return request, exists
}

func (socket *privateMessageWebsocket) removePendingRequest(id string) {
	pendingRequest, exists := socket.getPendingRequest(id)

	socket.pendingRequests.Mu.Lock()
	defer socket.pendingRequests.Mu.Unlock()

	if exists {
		delete(socket.pendingRequests.Map, id)
		close(pendingRequest.ch)
	}
}

func (socket *privateMessageWebsocket) sendChanThenDeleteWithTimeout(pendingRequest *pendingRequest, data []byte, timeout_sec int) {
	timeout_duration := time.Duration(timeout_sec) * time.Second

	pendingRequest.once.Do(
		func() {
			go func() {
				select {
				case pendingRequest.ch <- data:
					socket.removePendingRequest(pendingRequest.id)
				case <-time.After(timeout_duration):
					Logger.WARN("Private message channel send timeout")
					socket.removePendingRequest(pendingRequest.id)
				}
			}()
		},
	)
}

func (socket *privateMessageWebsocket) onError(err error) {
	if socket.OnError != nil {
		socket.OnError(err)
	}
}

func (socket *privateMessageWebsocket) onClose(code int, reason string) {
	if socket.OnClose != nil {
		socket.OnClose(code, reason)
	}
}

func (socket *privateMessageWebsocket) onMessage(msgType int, msg []byte) {
	requestId, isPrivate := CheckMessageIsPrivate(msg, socket.privateMessagePropertyName)
	if isPrivate {
		pendingRequest, exists := socket.getPendingRequest(requestId)
		if exists {
			socket.sendChanThenDeleteWithTimeout(pendingRequest, msg, 40)
		}

		if exists || !socket.isServer {
			return
		}
	}

	if socket.OnMessage != nil {
		socket.OnMessage(msgType, msg)
	}
}

// Public Methods

func (socket *privateMessageWebsocket) SendText(text string) error {
	return socket.base.SendText(text)
}

func (socket *privateMessageWebsocket) SendJSON(v interface{}) error {
	Logger.DEBUG(fmt.Sprintf("Sending JSON => %v", v))
	return socket.base.SendJSON(v)
}

func (socket *privateMessageWebsocket) SendPrivateMessage(message map[string]interface{}, timeout_sec ...int) (response []byte, hasTimedOut bool, err error) {
	request := socket.addPendingRequest()

	Logger.INFO(fmt.Sprintf("Sending private request of id %s => %v", request.id, message))

	message[socket.privateMessagePropertyName] = request.id

	err = socket.SendJSON(message)
	if err != nil {
		Logger.ERROR(fmt.Sprintf("[%s] There was an error sending JSON for private message", socket.base.url), err)
		return nil, false, err
	}

	timeout := 4
	if len(timeout_sec) > 0 {
		timeout = timeout_sec[0]
	}

	// Wait for response or timeout
	var timer <-chan time.Time
	if timeout > 0 {
		timer = time.After(time.Duration(timeout) * time.Second)
	}

	select {
	case resp := <-request.ch:
		return resp, false, nil
	case <-timer:
		request.once.Do(
			func() {
				socket.removePendingRequest(request.id)
			},
		)
		return nil, true, fmt.Errorf("the request has timed out after %d seconds", timeout)
	}
}

func (socket *privateMessageWebsocket) SendPreparedMessage(preparedMessage *ws.PreparedMessage) error {
	return socket.base.SendPreparedMessage(preparedMessage)
}

func (socket *privateMessageWebsocket) Close() {
	socket.base.Close()
}

//

func createPrivateMessageWebsocket(URL string, privateMessagePropertyName string, httpHeader http.Header, isServer bool) (*privateMessageWebsocket, error) {
	baseSocket, err := createBaseSocket(URL, httpHeader)
	if err != nil {
		return nil, err
	}

	var socket privateMessageWebsocket
	socket.init(baseSocket, privateMessagePropertyName, isServer)

	return &socket, nil
}

func assignPrivateMessageWebsocket(conn *ws.Conn, URL string, privateMessagePropertyName string, isServer bool) *privateMessageWebsocket {
	var socket privateMessageWebsocket

	baseSocket := assignBaseSocket(conn, URL)
	socket.init(baseSocket, privateMessagePropertyName, isServer)

	return &socket
}
