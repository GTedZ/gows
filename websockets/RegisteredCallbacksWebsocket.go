package websockets

import (
	"net/http"

	"github.com/GTedZ/gows/parser"
	ws "github.com/gorilla/websocket"
)

type RegisteredCallbacksWebsocket struct {
	base *privateMessageWebsocket

	parserRegistry *parser.MessageParsers_Registry

	OnMessage func(messageType int, msg []byte)
	OnError   func(err error)
	OnClose   func(code int, reason string)
}

func (socket *RegisteredCallbacksWebsocket) init(baseSocket *privateMessageWebsocket) {
	socket.base = baseSocket
	socket.parserRegistry = &parser.MessageParsers_Registry{}

	socket.base.OnMessage = socket.onMessage
	socket.base.OnError = socket.onError
	socket.base.OnClose = socket.onClose
}

func (socket *RegisteredCallbacksWebsocket) onMessage(messageType int, msg []byte) {
	if socket.parserRegistry.TryDispatch(msg) {
		return
	}

	if socket.OnMessage != nil {
		socket.OnMessage(messageType, msg)
	}
}

func (socket *RegisteredCallbacksWebsocket) onError(err error) {
	if socket.OnError != nil {
		socket.OnError(err)
	}
}

func (socket *RegisteredCallbacksWebsocket) onClose(code int, reason string) {
	if socket.OnClose != nil {
		socket.OnClose(code, reason)
	}
}

////

//// Public Methods

func (socket *RegisteredCallbacksWebsocket) GetParserRegistry() *parser.MessageParsers_Registry {
	return socket.parserRegistry
}

//

func (socket *RegisteredCallbacksWebsocket) SendText(text string) error {
	return socket.base.SendText(text)
}

func (socket *RegisteredCallbacksWebsocket) SendJSON(v interface{}) error {
	return socket.base.SendJSON(v)
}

func (socket *RegisteredCallbacksWebsocket) SendPrivateMessage(message map[string]interface{}, timeout_sec ...int) (response []byte, hasTimedOut bool, err error) {
	return socket.base.SendPrivateMessage(message, timeout_sec...)
}

func (socket *RegisteredCallbacksWebsocket) SendPreparedMessage(preparedMessage *ws.PreparedMessage) error {
	return socket.base.SendPreparedMessage(preparedMessage)
}

func (socket *RegisteredCallbacksWebsocket) Close() {
	socket.base.Close()
}

////

func CreateRegisteredCallbacksWebsocket(URL string, privateMessagePropertyName string, isServer bool, httpHeader http.Header) (*RegisteredCallbacksWebsocket, error) {
	var socket RegisteredCallbacksWebsocket

	baseSocket, err := createPrivateMessageWebsocket(URL, privateMessagePropertyName, httpHeader, isServer)
	if err != nil {
		return nil, err
	}

	socket.init(baseSocket)

	return &socket, nil
}

func AssignRegisteredCallbacksWebsocket(conn *ws.Conn, URL string, privateMessagePropertyName string, isServer bool) *RegisteredCallbacksWebsocket {
	var socket RegisteredCallbacksWebsocket

	baseSocket := assignPrivateMessageWebsocket(conn, URL, privateMessagePropertyName, isServer)

	socket.init(baseSocket)

	return &socket
}
