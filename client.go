package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	playerID string
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(5120)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("read error: %v", err)
			}
			break
		}
		var env Envelope
		if err := json.Unmarshal(message, &env); err != nil {
			log.Printf("invalid message: %v", err)
			continue
		}
		switch env.Type {
		case "boop_request":
			var payload struct {
				BoopedID string `json:"boopedID"`
			}
			if err := json.Unmarshal(env.Payload, &payload); err != nil {
				log.Printf("bad boop payload: %v", err)
				continue
			}
			if payload.BoopedID == "" {
				continue
			}
			// Update state (store latest direction only), reject duplicate
			if changed := c.hub.game.RecordBoop(c.playerID, payload.BoopedID); changed {
				boopEvent := map[string]any{
					"booperID": c.playerID,
					"boopedID": payload.BoopedID,
				}
				msg := mustJSON(map[string]any{
					"type":    "boop_event",
					"payload": boopEvent,
				})
				c.hub.Broadcast(msg)

				// Log the boop as a single-line JSON object for observability
				logEntry := map[string]any{
					"ts":       time.Now().UTC().Format(time.RFC3339Nano),
					"event":    "boop",
					"booperID": c.playerID,
					"boopedID": payload.BoopedID,
				}
				if b, err := json.Marshal(logEntry); err == nil {
					log.Println(string(b))
				} else {
					log.Printf("failed to marshal boop log: %v", err)
				}
			} else {
				// Notify sender their boop was denied due to duplicate
				deny := mustJSON(map[string]any{
					"type": "boop_denied",
					"payload": map[string]any{
						"boopedID": payload.BoopedID,
						"reason":   "already-booped",
					},
				})
				select {
				case c.send <- deny:
				default:
				}
			}
		default:
			// ignore unknown types for now
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// channel closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
