package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/livepoll/backend/internal/models"
	"github.com/livepoll/backend/internal/repository"
	"github.com/redis/go-redis/v9"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow any origin for easy cross-origin live streaming & previewing
		return true
	},
}

// Client represents a single connected browser WebSocket
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	pollID   string
	isClosed bool
	mu       sync.Mutex
}

// Room holds all clients connected to a specific poll
type Room struct {
	pollID     string
	clients    map[*Client]bool
	pubsub     *redis.PubSub
	cancelSub  context.CancelFunc
	mu         sync.RWMutex
}

// Hub maintains active rooms and fan-out
type Hub struct {
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	redisRepo  *repository.RedisRepo
	mu         sync.RWMutex
}

func NewHub(redisRepo *repository.RedisRepo) *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		redisRepo:  redisRepo,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)

		case client := <-h.unregister:
			h.removeClient(client)
		}
	}
}

func (h *Hub) addClient(client *Client) {
	h.mu.Lock()
	room, exists := h.rooms[client.pollID]
	if !exists {
		room = &Room{
			pollID:  client.pollID,
			clients: make(map[*Client]bool),
		}
		h.rooms[client.pollID] = room

		// Start Redis Pub/Sub or In-Memory subscription for this poll room
		ctx, cancel := context.WithCancel(context.Background())
		room.cancelSub = cancel
		if h.redisRepo.IsInMemory() {
			memCh := make(chan []byte, 32)
			unsubscribe := h.redisRepo.SubscribeInMemory(client.pollID, memCh)
			go func() {
				defer unsubscribe()
				for {
					select {
					case <-ctx.Done():
						return
					case msg, ok := <-memCh:
						if !ok {
							return
						}
						h.broadcastToRoom(room, msg)
					}
				}
			}()
		} else {
			room.pubsub = h.redisRepo.SubscribeToPoll(ctx, client.pollID)
			go h.listenRedisChannel(room, ctx)
		}
	}
	h.mu.Unlock()

	room.mu.Lock()
	room.clients[client] = true
	room.mu.Unlock()

	log.Printf("[WS] Client connected to poll room: %s (Total in room: %d)", client.pollID, len(room.clients))
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	room, exists := h.rooms[client.pollID]
	if !exists {
		h.mu.Unlock()
		return
	}

	room.mu.Lock()
	if _, ok := room.clients[client]; ok {
		delete(room.clients, client)
		client.close()
	}
	empty := len(room.clients) == 0
	room.mu.Unlock()

	if empty {
		// Clean up Redis/Memory subscription and room to free resources
		if room.cancelSub != nil {
			room.cancelSub()
		}
		if room.pubsub != nil {
			_ = room.pubsub.Close()
		}
		delete(h.rooms, client.pollID)
		log.Printf("[WS] All clients left poll room %s, closed subscription", client.pollID)
	}
	h.mu.Unlock()
}

func (h *Hub) listenRedisChannel(room *Room, ctx context.Context) {
	if room.pubsub == nil {
		return
	}
	ch := room.pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			h.broadcastToRoom(room, []byte(msg.Payload))
		}
	}
}

func (h *Hub) broadcastToRoom(room *Room, message []byte) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	for client := range room.clients {
		select {
		case client.send <- message:
		default:
			// Buffer full, drop client
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}
}

// BroadcastDirect sends an update directly to a room (e.g., when Redis is unavailable or on initial connect)
func (h *Hub) BroadcastDirect(pollID string, update *models.LivePollUpdate) {
	data, err := json.Marshal(update)
	if err != nil {
		return
	}

	h.mu.RLock()
	room, exists := h.rooms[pollID]
	h.mu.RUnlock()

	if exists {
		h.broadcastToRoom(room, data)
	}
}

func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isClosed {
		c.isClosed = true
		close(c.send)
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Read error: %v", err)
			}
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWS handles incoming websocket upgrade requests
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, pollID string, initialState *models.LivePollUpdate) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return err
	}

	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 64),
		pollID: pollID,
	}

	hub.register <- client

	// Push initial state immediately
	if initialState != nil {
		if data, err := json.Marshal(initialState); err == nil {
			client.send <- data
		}
	}

	go client.writePump()
	go client.readPump()

	return nil
}
