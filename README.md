# Go-WS: A High-Level WebSocket Library for Go

Go-WS is a wrapper built around [Gorilla WebSocket](https://github.com/gorilla/websocket), designed to simplify and accelerate WebSocket communication for both server and client implementations in Go.  
It introduces high-level event emitters, request-response abstractions, and message parsing logic, making it ideal for building scalable real-time applications.


## Features

- Easy WebSocket server and client setup
- Event-driven architecture (connection, disconnection, errors, messages)
- Request-response pattern built-in
- **Custom message parsing and typed handler registration**
- Graceful connection handling with ID assignment
- Thread-safe message sending
- TLS certificate support (generate/load/save)
- Reconnectable clients


## Installation

```bash
go get github.com/GTedZ/gows
```

## Getting Started

### 1. Setting Up a WebSocket Server

```go
server := gows.NewServer("0.0.0.0", "/ws")
err := server.ListenAndServe(3000)
if err != nil {
    fmt.Println("ListenAndServe error", err.Error())
}
```

---

### 2. Configuring Origin Check (optional)

```go
server.SetCheckOrigin(
    func(r *http.Request) bool {
        return true // Allow all origins
    },
)
```

### Or just allow all (default):

```go
server.SetCheckOrigin(nil)
```

---

### 3. Handling Events

```go
server.OnConnect = func(conn *gows.Connection) {
    fmt.Println("New client connected")
}

server.OnClose = func(conn *gows.Connection, code int, reason string) {
    fmt.Println("Client disconnected:", reason)
}
```

---

### 4. Connection-Level Metadata (Custom Key-Value Store)
Each connection holds a thread-safe key-value store:

```go
conn.SetString("username", "alice")
user, exists := conn.GetString("username")
```

---

### 5. Sending Text or JSON Messages

```go
conn.SendText("Hello Client!")

conn.SendJSON(struct {
    Event string
    Data  string
}{
    Event: "greeting",
    Data:  "Hello!",
})
```

## Websocket Client

### 1. Connecting to a Server

```go
client, err := gows.NewClient("ws://localhost:3000/ws")
if err != nil {
    panic(err)
}
```

---

### 2. Receiving Messages

```go
client.OnMessage = func(messageType int, msg []byte) {
    fmt.Println("Received:", string(msg))
}
```

---

### 3. 🔄 Request-Response Pattern

```go
// Client
req := map[string]interface{}{"method": "ping"}
resp, timedOut, err := client.SendPrivateMessage(req)
```

```go
// Server
conn.OnRequest = func(msg []byte, req *gows.ResponseHandler) {
    res := map[string]interface{}{"reply": "pong"}
    req.Reply(res)
}
```
- Both client and server can initiate requests.
- Uses a private field like "id" to match requests/responses.
- Customize this field name using SetRequestIdPropertyName().

---

### 4. 🧠 Message Parsers (Core Feature)

Typed dispatching of messages using JSON structure matching:

#### Example Type

```go
type Candlestick struct {
    Timestamp int64
    Open      float64
    High      float64
    Low       float64
    Close     float64
    Volume    float64
}
```

#### Registering a Parser on Client or Server

```go
gows.RegisterMessageParserCallback(
    client.GetParserRegistry(),

    func(b []byte) (bool, *Candlestick) {   // Parser function, make sure to be overly explicit, a <nil> error from Unmarshal isn't enough
        var c Candlestick
        err := json.Unmarshal(b, &c)
        return err == nil && c.Timestamp > 0, &c
    },

    func(c *Candlestick) {                  // Callback
        fmt.Println("Parsed candlestick:", c)
    },
)
```

#### Parser Function Format

```go
func(b []byte) (bool, *YourType)
```

- bool: whether this message matches
- *YourType: parsed struct to forward to the callback

Alternatively, pass nil for the parser to use default json.Unmarshal.

---

### 5. 🔐 TLS Support

#### Generate Self-Signed Certificate

```go
cert, certPEM, keyPEM, _ := gows.Certificates.GenerateSelfSignedCert("localhost")
gows.Certificates.SaveCertToFiles(certPEM, keyPEM, "cert.pem", "key.pem")
```

#### Start TLS Server

```go
server.ListenAndServeTLS(443, cert)
```

Or load certs from files:

```go
server.ListenAndServeTLSWithFiles(443, "cert.pem", "key.pem")
```

| Event         | Triggered When                      |
| ------------- | ----------------------------------- |
| `OnConnect`   | A new client connects               |
| `OnClose`     | A client disconnects                |
| `OnMessage`   | Any message received                |
| `OnRequest`   | A private message (request) arrives |
| `OnReconnect` | Client successfully reconnects      |
| `OnError`     | Any socket-level error occurs       |


## Example: Echo Server and Client

### Server

```go
server := gows.NewServer("0.0.0.0", "/ws")
server.OnConnect = func(conn *gows.Connection) {
    conn.OnMessage = func(_ int, msg []byte) {
        conn.SendText("Echo: " + string(msg))
    }
}
server.ListenAndServe(9000)
```

### Client
```go
client, _ := gows.NewClient("ws://localhost:9000/ws")
client.OnMessage = func(_ int, msg []byte) {
    fmt.Println("Reply:", string(msg))
}
client.SendText("Hello!")
```

---

## Contributions

Feel free to submit issues and pull requests. Feedback and improvements are welcome.

## Support

If you would like to support my projects, you can donate to the following addresses:

BTC: `19XC9b9gkbNVTB9eTPzjByijmx8gNd2qVR`

ETH (ERC20): `0x7e5db6e135da1f4d5de26f7f7d4ed10dada20478`

USDT (TRC20): `TLHo8zxBpFSTdyuvvCTG4DzXFfDfv49EMu`

SOL: `B3ZKvA3yCm5gKLb7vucU7nu8RTnp6MEAPpfeySMSMoDc`

Thank you for your support!

---