package websockets

import (
	"fmt"
	"math"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/GTedZ/gows/parser"
	ws "github.com/gorilla/websocket"
)

type ReconnectingRegisteredCallbacksWebsocket struct {
	base                       *RegisteredCallbacksWebsocket
	privateMessagePropertyName string

	url        string
	isServer   bool
	httpHeader http.Header
	ready      atomic.Bool
	closed     atomic.Bool

	OnMessage        func(messageType int, msg []byte)
	OnError          func(err error)
	OnDisconnect     func(code int, reason string)
	OnReconnectError func(err error)
	OnReconnect      func()
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) init(URL string, privateMessagePropertyName string, isServer bool, httpHeader http.Header) {
	socket.url = URL
	socket.privateMessagePropertyName = privateMessagePropertyName
	socket.isServer = isServer
	socket.httpHeader = httpHeader
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) init_subsocket(subsocket *RegisteredCallbacksWebsocket) {
	socket.ready.Store(false)

	socket.base = subsocket

	socket.base.OnMessage = socket.onMessage
	socket.base.OnError = socket.onError
	socket.base.OnClose = socket.onSubsocketClosed

	socket.ready.Store(true)
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) onSubsocketClosed(code int, reason string) {
	// The reconnecting socket has been terminally closed, so no need to reconnect or do anything
	if socket.closed.Load() {
		return
	}

	go func() {
		socket.ready.Store(false)
		if socket.OnDisconnect != nil {
			socket.OnDisconnect(code, reason)
		}

		err := socket.newSubsocket(true)
		if err != nil {
			return
		}

		if socket.OnReconnect != nil {
			socket.OnReconnect()
		}
	}()
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) newSubsocket(retry bool) error {
	socket.ready.Store(false)

	var newSocket *RegisteredCallbacksWebsocket
	var err error
	var retries int = 0
	for {
		if socket.closed.Load() {
			return fmt.Errorf("socket manually closed")
		}

		Logger.DEBUG(fmt.Sprintf("[%s] Connecting to new subsocket", socket.url))
		newSocket, err = CreateRegisteredCallbacksWebsocket(socket.url, socket.privateMessagePropertyName, socket.isServer, socket.httpHeader)
		if err != nil {
			Logger.ERROR(fmt.Sprintf("[%s] Failed to open subsocket", socket.url), err)

			if !retry {
				return err
			} else {
				if socket.OnReconnectError != nil {
					socket.OnReconnectError(err)
				}
			}

			extraDuration := int(math.Min(float64(retries*250), 2000))
			duration := time.Duration(500 + extraDuration)
			time.Sleep(duration * time.Millisecond)
			retries++
			continue
		}
		break
	}

	socket.init_subsocket(newSocket)

	return nil
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) onMessage(messageType int, msg []byte) {
	if socket.OnMessage != nil {
		socket.OnMessage(messageType, msg)
	}
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) onError(err error) {
	if socket.OnError != nil {
		socket.OnError(err)
	}
}

//// Public methods

func (socket *ReconnectingRegisteredCallbacksWebsocket) SetURL(URL string) {
	socket.url = URL
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) SetHTTPHeader(httpHeader http.Header) {
	socket.httpHeader = httpHeader
}

//

func (socket *ReconnectingRegisteredCallbacksWebsocket) GetParserRegistry() *parser.MessageParsers_Registry {
	return socket.base.GetParserRegistry()
}

//

func (socket *ReconnectingRegisteredCallbacksWebsocket) SendText(text string) error {
	return socket.base.SendText(text)
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) SendJSON(v interface{}) error {
	return socket.base.SendJSON(v)
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) SendPrivateMessage(message map[string]interface{}, timeout_sec ...int) (response []byte, hasTimedOut bool, err error) {
	for {
		if socket.closed.Load() {
			return nil, false, fmt.Errorf("socket has been closed")
		}
		if !socket.ready.Load() {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}
	return socket.base.SendPrivateMessage(message, timeout_sec...)
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) SendPreparedMessage(preparedMessage *ws.PreparedMessage) error {
	return socket.base.SendPreparedMessage(preparedMessage)
}

func (socket *ReconnectingRegisteredCallbacksWebsocket) Close() {
	socket.closed.Store(true)

	socket.base.Close()
}

////

// WARNING: Since this is a reconnecting socket, it will NEVER error out when trying to connect to the server, it will block until a successful connection is established
func CreateReconnectingRegisteredCallbacksWebsocket(URL string, privateMessagePropertyName string, isServer bool, httpHeader http.Header) (*ReconnectingRegisteredCallbacksWebsocket, error) {
	var socket ReconnectingRegisteredCallbacksWebsocket

	socket.init(URL, privateMessagePropertyName, isServer, httpHeader)

	err := socket.newSubsocket(false)
	if err != nil {
		return nil, err
	}

	return &socket, nil
}

func AssignReconnectingRegisteredCallbacksWebsocket(conn *ws.Conn, URL string, privateMessagePropertyName string, isServer bool, httpHeader http.Header) *ReconnectingRegisteredCallbacksWebsocket {
	var socket ReconnectingRegisteredCallbacksWebsocket

	socket.init(URL, privateMessagePropertyName, isServer, httpHeader)

	baseSocket := AssignRegisteredCallbacksWebsocket(conn, URL, privateMessagePropertyName, isServer)
	socket.init_subsocket(baseSocket)

	return &socket
}
