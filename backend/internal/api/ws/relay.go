package ws

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Relay handles WebSocket connections from the frontend and proxies them to mihomo
type Relay struct {
	mihomoAddr string
	secret     string
}

func NewRelay(mihomoAddr, secret string) *Relay {
	return &Relay{
		mihomoAddr: mihomoAddr,
		secret:     secret,
	}
}

// HandleTraffic relays traffic data from mihomo to frontend clients
func (r *Relay) HandleTraffic(w http.ResponseWriter, req *http.Request) {
	r.relay(w, req, "/traffic")
}

// HandleConnections relays connection data from mihomo to frontend clients
func (r *Relay) HandleConnections(w http.ResponseWriter, req *http.Request) {
	interval := req.URL.Query().Get("interval")
	if interval == "" {
		interval = "1000"
	}
	path := fmt.Sprintf("/connections?interval=%s", interval)
	r.relay(w, req, path)
}

// HandleMemory relays memory data from mihomo to frontend clients
func (r *Relay) HandleMemory(w http.ResponseWriter, req *http.Request) {
	r.relay(w, req, "/memory")
}

// relay upgrades the HTTP connection to WebSocket, connects to mihomo, and relays messages
func (r *Relay) relay(w http.ResponseWriter, req *http.Request, mihomoPath string) {
	// Upgrade frontend connection
	frontendConn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}
	defer frontendConn.Close()

	// Connect to mihomo WebSocket
	mihomoURL := url.URL{
		Scheme: "ws",
		Host:   r.mihomoAddr,
		Path:   mihomoPath,
	}

	header := http.Header{}
	if r.secret != "" {
		header.Set("Authorization", "Bearer "+r.secret)
	}

	backendConn, resp, err := websocket.DefaultDialer.Dial(mihomoURL.String(), header)
	if err != nil {
		log.Printf("ws connect to mihomo failed: %v (resp: %v)", err, resp)
		// Send error to frontend
		frontendConn.WriteJSON(map[string]string{"error": "failed to connect to mihomo"})
		return
	}
	defer backendConn.Close()

	// Bidirectional relay
	var wg sync.WaitGroup
	wg.Add(2)

	// Backend → Frontend
	go func() {
		defer wg.Done()
		for {
			msgType, msg, err := backendConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("ws backend read error: %v", err)
				}
				return
			}
			if err := frontendConn.WriteMessage(msgType, msg); err != nil {
				log.Printf("ws frontend write error: %v", err)
				return
			}
		}
	}()

	// Frontend → Backend (for commands like pause/resume)
	go func() {
		defer wg.Done()
		for {
			msgType, msg, err := frontendConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("ws frontend read error: %v", err)
				}
				return
			}
			if err := backendConn.WriteMessage(msgType, msg); err != nil {
				log.Printf("ws backend write error: %v", err)
				return
			}
		}
	}()

	wg.Wait()
}
