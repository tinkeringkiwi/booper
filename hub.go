package main

import (
	"encoding/json"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	game       *Game
}

func NewHub(game *Game) *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
		// Buffered broadcast channel to allow short bursts without dropping.
		// Size tuned to handle spikes; increase if you expect larger bursts.
		broadcast:  make(chan []byte, 1024),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		game:       game,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// Remove player from game state
				if client.playerID != "" {
					h.game.RemovePlayer(client.playerID)
					// Broadcast player_left to remaining clients
					payload := map[string]any{"id": client.playerID}
					msg, err := json.Marshal(map[string]any{
						"type":    "player_left",
						"payload": payload,
					})
					if err == nil {
						for c := range h.clients {
							select {
							case c.send <- msg:
							default:
								close(c.send)
								delete(h.clients, c)
							}
						}
					}
				}
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *Hub) Broadcast(data []byte) {
	// Use a blocking send into the buffered broadcast channel. This will
	// wait briefly if the channel is full, but avoid silently dropping
	// messages. The buffer size prevents most spikes from blocking.
	h.broadcast <- data
}
