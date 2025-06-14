package gows

import (
	"sync"

	"github.com/GTedZ/gows/parser"
	"github.com/GTedZ/gows/websockets"
	ws "github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

type Connection struct {
	base   *websockets.RegisteredCallbacksWebsocket
	parent *Server

	connectionId int

	data struct {
		mu         sync.Mutex
		ints       map[string]int64
		floats     map[string]float64
		strings    map[string]string
		interfaces map[string]interface{}
	}

	OnRequest func(msg []byte, request *ResponseHandler)
	OnMessage func(messageType int, msg []byte)
	OnError   func(err error)
	OnClose   func(code int, reason string)
}

func (connection *Connection) init(parent *Server, conn *ws.Conn, privateMessagePropertyName string, connectionId int) {
	connection.base = websockets.AssignRegisteredCallbacksWebsocket(conn, "", privateMessagePropertyName, true)
	connection.parent = parent
	connection.connectionId = connectionId

	connection.data.ints = make(map[string]int64)
	connection.data.floats = make(map[string]float64)
	connection.data.strings = make(map[string]string)
	connection.data.interfaces = make(map[string]interface{})

	//

	connection.base.OnMessage = connection.onMessage
	connection.base.OnError = connection.onError
	connection.base.OnClose = connection.onClose
}

func (connection *Connection) onMessage(messageType int, msg []byte) {
	requestId, isRequest := websockets.CheckMessageIsPrivate(msg, connection.parent.privateMessagePropertyName)
	if isRequest {
		connection.onRequest(requestId, msg)
		return
	}

	if connection.OnMessage != nil {
		connection.OnMessage(messageType, msg)
	}
}

func (connection *Connection) onError(err error) {
	if connection.OnError != nil {
		connection.OnError(err)
	}
}

func (connection *Connection) onClose(code int, reason string) {
	connection.parent.onConnectionClose(connection, code, reason)

	if connection.OnClose != nil {
		connection.OnClose(code, reason)
	}
}

//

type ResponseHandler struct {
	parent *Connection

	requestPropertyName string
	requestId           string

	Body []byte
}

func (request *ResponseHandler) init(connection *Connection, requestPropertyName string, requestId string, body []byte) {
	request.parent = connection
	request.requestPropertyName = requestPropertyName
	request.requestId = requestId

	request.Body = body
}

func (request *ResponseHandler) Unmarshal(v interface{}) error {
	return jsoniter.Unmarshal(request.Body, v)
}

func (request *ResponseHandler) Reply(reply map[string]interface{}) error {
	reply[request.requestPropertyName] = request.requestId

	return request.parent.SendJSON(reply)
}

func (connection *Connection) onRequest(requestId string, msg []byte) {
	var request ResponseHandler
	request.init(connection, connection.parent.privateMessagePropertyName, requestId, msg)

	if connection.OnRequest != nil {
		connection.OnRequest(msg, &request)
	}
}

//// Public Methods

func (connection *Connection) GetId() int {
	return connection.connectionId
}

//

// Used to get an int from a key-value store unique to each connection
//
// NOTE: ints are stored independently, so you can use the same key for different types
func (connection *Connection) GetInt(key string) (value int64, exists bool) {
	connection.data.mu.Lock()
	defer connection.data.mu.Unlock()

	value, exists = connection.data.ints[key]
	return value, exists
}

// Used to set an int to a key-value store unique to each connection
//
// NOTE: ints are stored independently, so you can use the same key for different types
func (connection *Connection) SetInt(key string, value int64) {
	connection.data.mu.Lock()
	defer connection.data.mu.Unlock()

	connection.data.ints[key] = value
}

//

// Used to get a float from a key-value store unique to each connection
//
// NOTE: floats are stored independently, so you can use the same key for different types
func (connection *Connection) GetFloat(key string) (value float64, exists bool) {
	connection.data.mu.Lock()
	defer connection.data.mu.Unlock()

	value, exists = connection.data.floats[key]
	return value, exists
}

// Used to set a float to a key-value store unique to each connection
//
// NOTE: floats are stored independently, so you can use the same key for different types
func (connection *Connection) SetFloat(key string, value float64) {
	connection.data.mu.Lock()
	defer connection.data.mu.Unlock()

	connection.data.floats[key] = value
}

//

// Used to get a string from a key-value store unique to each connection
//
// NOTE: interfaces and strings are different underlying maps, so you can use the same key for each different type
func (connection *Connection) GetString(key string) (value string, exists bool) {
	connection.data.mu.Lock()
	defer connection.data.mu.Unlock()

	value, exists = connection.data.strings[key]
	return value, exists
}

// Used to set a string to a key-value store unique to each connection
//
// NOTE: interfaces and strings are different underlying maps, so you can use the same key for each different type
func (connection *Connection) SetString(key string, value string) {
	connection.data.mu.Lock()
	defer connection.data.mu.Unlock()

	connection.data.strings[key] = value
}

//

// Used to get an interface from a key-value store unique to each connection
//
// NOTE: interfaces and strings are different underlying maps, so you can use the same key for each different type
func (connection *Connection) GetInterface(key string) (value interface{}, exists bool) {
	connection.data.mu.Lock()
	defer connection.data.mu.Unlock()

	value, exists = connection.data.interfaces[key]
	return value, exists
}

// Used to set an interface to a key-value store unique to each connection
//
// NOTE: interfaces and strings are different underlying maps, so you can use the same key for each different type
func (connection *Connection) SetInterface(key string, value interface{}) {
	connection.data.mu.Lock()
	defer connection.data.mu.Unlock()

	connection.data.interfaces[key] = value
}

//

func (connection *Connection) GetParserRegistry() *parser.MessageParsers_Registry {
	return connection.base.GetParserRegistry()
}

//

func (connection *Connection) SendText(text string) error {
	return connection.base.SendText(text)
}

func (connection *Connection) SendJSON(v interface{}) error {
	return connection.base.SendJSON(v)
}

func (connection *Connection) SendPreparedMessage(message *ws.PreparedMessage) error {
	return connection.base.SendPreparedMessage(message)
}

func (connection *Connection) Close() {
	connection.base.Close()
}

////

func assignConnection(parent *Server, conn *ws.Conn, privateMessagePropertyName string, connectionId int) *Connection {
	var connection Connection

	connection.init(parent, conn, privateMessagePropertyName, connectionId)

	return &connection
}
