package gows

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

type Server_Params struct {
	PrivateMessagePropertyName string
}

type Server struct {
	upgrader                   websocket.Upgrader
	addr                       string
	path                       string
	privateMessagePropertyName string
	nextConnectionId           int

	OnConnect func(*Connection)
	OnClose   func(connection *Connection, code int, reason string)

	Connections struct {
		Mu  sync.Mutex
		Map map[int]*Connection
	}
}

func (server *Server) init(addr string, path string, privateMessagePropertyName string) {
	server.addr = addr
	server.path = path
	server.privateMessagePropertyName = privateMessagePropertyName

	server.SetCheckOrigin(nil)

	server.Connections.Map = make(map[int]*Connection)
}

func (server *Server) onConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := server.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrade error:", err)
		return
	}

	connectionId := server.nextConnectionId
	server.nextConnectionId++

	connection := assignConnection(server, conn, server.privateMessagePropertyName, connectionId)

	server.addConnection(connection)

	if server.OnConnect != nil {
		server.OnConnect(connection)
	}
}

func (server *Server) addConnection(connection *Connection) {
	server.Connections.Mu.Lock()
	defer server.Connections.Mu.Unlock()

	server.Connections.Map[connection.GetId()] = connection
}

func (server *Server) removeConnection(connection *Connection) {
	server.Connections.Mu.Lock()
	defer server.Connections.Mu.Unlock()

	delete(server.Connections.Map, connection.GetId())
}

//

func (server *Server) onConnectionClose(connection *Connection, code int, reason string) {
	server.removeConnection(connection)

	if server.OnClose != nil {
		server.OnClose(connection, code, reason)
	}
}

////

// CheckOrigin returns true if the request Origin header is acceptable. If CheckOrigin is nil, then a safe default is used: return false if the Origin request header is present and the origin host is not equal to request Host header.
//
// A CheckOrigin function should carefully validate the request origin to prevent cross-site request forgery.
func (server *Server) SetCheckOrigin(checkOrigin func(r *http.Request) bool) {
	if checkOrigin == nil {
		server.upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
	} else {
		server.upgrader.CheckOrigin = checkOrigin
	}
}

// This method WILL block until the server is terminated or an error occurs
func (server *Server) ListenAndServe(port int) error {
	http.HandleFunc(server.path, server.onConnect)
	fullAddr := fmt.Sprintf("%s:%d", server.addr, port)
	return http.ListenAndServe(fullAddr, nil)
}

// This method WILL block until the server is terminated or an error occurs
func (server *Server) ListenAndServeTLS(port int, certificate tls.Certificate) error {
	http.HandleFunc(server.path, server.onConnect)
	fullAddr := fmt.Sprintf("%s:%d", server.addr, port)

	s := &http.Server{
		Addr: fullAddr,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{certificate},
		},
	}

	return s.ListenAndServeTLS("", "")
}

// This method WILL block until the server is terminated or an error occurs
func (server *Server) ListenAndServeTLSWithFiles(port int, certFile, keyFile string) error {
	http.HandleFunc(server.path, server.onConnect)
	fullAddr := fmt.Sprintf("%s:%d", server.addr, port)
	return http.ListenAndServeTLS(fullAddr, certFile, keyFile, nil)
}

// Broadcasts a message to all active connections to the server
//
// 'err' is returned only when preparing the message for broadcast goes wrong, meaning no connection was sent the message
//
// Otherwise 'failCount' keeps track of how many connections failed to be sent the message
func (server *Server) Broadcast(v interface{}) (failCount int, err error) {
	data, err := jsoniter.Marshal(v)
	if err != nil {
		return 0, err
	}

	preparedMessage, err := websocket.NewPreparedMessage(websocket.TextMessage, data)
	if err != nil {
		return 0, err
	}

	server.Connections.Mu.Lock()
	defer server.Connections.Mu.Unlock()

	for _, connection := range server.Connections.Map {
		err := connection.SendPreparedMessage(preparedMessage)
		if err != nil {
			failCount++
		}
	}

	return failCount, nil
}

////

func NewServer(addr string, path string, opt_params ...Server_Params) *Server {
	var server Server

	privateMessagePropertyName := "id"
	if len(opt_params) != 0 {
		params := opt_params[0]

		if params.PrivateMessagePropertyName != "" {
			privateMessagePropertyName = params.PrivateMessagePropertyName
		}
	}

	server.init(addr, path, privateMessagePropertyName)

	return &server
}
