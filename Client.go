package gows

import (
	"net/http"
	"net/url"

	"github.com/GTedZ/gows/parser"
	"github.com/GTedZ/gows/websockets"
	ws "github.com/gorilla/websocket"
)

type Client_params struct {
	query   map[string]string
	headers http.Header
	// Default is "id"
	PrivateRequestPropertyName string
}

type Client struct {
	base *websockets.ReconnectingRegisteredCallbacksWebsocket

	OnMessage func(messageType int, msg []byte)

	// Called on any error that originates from the current established connection.
	//
	// Do not treat this as a disconnection as it can be triggerred multiple times for the same connection and is not a sign of disconnection
	//
	// Use 'OnDisconnect' instead in that case
	OnError func(err error)

	// Called once the connection unexpectedly drops, expect to be reconnected shortly after
	OnDisconnect func(code int, reason string)

	OnReconnectError func(err error)
	// Called once a new connection is established after disconnection
	OnReconnect func()
}

func (socket *Client) init(baseSocket *websockets.ReconnectingRegisteredCallbacksWebsocket) {
	socket.base = baseSocket

	socket.base.OnMessage = socket.onMessage
	socket.base.OnError = socket.onError
	socket.base.OnDisconnect = socket.onDisconnect
	socket.base.OnReconnectError = socket.onReconnectError
	socket.base.OnReconnect = socket.onReconnect
}

func (socket *Client) onMessage(messageType int, msg []byte) {
	if socket.OnMessage != nil {
		socket.OnMessage(messageType, msg)
	}
}

func (socket *Client) onError(err error) {
	if socket.OnError != nil {
		socket.OnError(err)
	}
}

func (socket *Client) onDisconnect(code int, reason string) {
	if socket.OnDisconnect != nil {
		socket.OnDisconnect(code, reason)
	}
}

func (socket *Client) onReconnectError(err error) {
	if socket.OnReconnectError != nil {
		socket.OnReconnectError(err)
	}
}

func (socket *Client) onReconnect() {
	if socket.OnReconnect != nil {
		socket.OnReconnect()
	}
}

//// Public Methods

func (socket *Client) SetURL(URL string) {
	socket.base.SetURL(URL)
}

func (socket *Client) SetHTTPHeader(httpHeader http.Header) {
	socket.base.SetHTTPHeader(httpHeader)
}

//

func (socket *Client) GetParserRegistry() *parser.MessageParsers_Registry {
	return socket.base.GetParserRegistry()
}

//

func (socket *Client) SendText(text string) error {
	return socket.base.SendText(text)
}

func (socket *Client) SendJSON(v interface{}) error {
	return socket.base.SendJSON(v)
}

func (socket *Client) SendPrivateMessage(message map[string]interface{}, timeout_sec ...int) (response []byte, hasTimedOut bool, err error) {
	return socket.base.SendPrivateMessage(message, timeout_sec...)
}

func (socket *Client) SendPreparedMessage(preparedMessage *ws.PreparedMessage) error {
	return socket.base.SendPreparedMessage(preparedMessage)
}

func (socket *Client) Close() {
	socket.base.Close()
}

////

func NewClient(URL string, opt_params ...Client_params) (*Client, error) {
	var headers http.Header
	var privateRequestPropertyName = "id"

	if len(opt_params) != 0 {
		params := opt_params[0]

		// Preparing the URL
		u, err := url.Parse(URL)
		if err != nil {
			return nil, err
		}

		q := u.Query()
		for key, value := range params.query {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()

		URL = u.String()

		// Handling headers
		if params.headers != nil {
			headers = params.headers
		}

		// Handling private request property name
		if params.PrivateRequestPropertyName != "" {
			privateRequestPropertyName = params.PrivateRequestPropertyName
		}
	}

	baseSocket, err := websockets.CreateReconnectingRegisteredCallbacksWebsocket(URL, privateRequestPropertyName, false, headers)
	if err != nil {
		return nil, err
	}

	var socket Client
	socket.init(baseSocket)

	return &socket, err
}
