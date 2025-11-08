package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func HandleRoot(game *Game) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join("frontend", "templates", "index.gohtml")
		// Always parse on each request (small file; ensures latest changes visible during dev)
		tmpl, err := template.ParseFiles(p)
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		if err := tmpl.Execute(w, nil); err != nil {
			log.Printf("template execute: %v", err)
		}
	}
}

func HandleStatic() http.Handler {
	fs := http.FileServer(http.Dir(filepath.Join("frontend", "static")))
	return http.StripPrefix("/static/", fs)
}

func HandleWebSocket(hub *Hub, game *Game) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade: %v", err)
			return
		}
		player := NewPlayer()
		game.AddPlayer(player)

		// Increase per-client send buffer to reduce client-side drops during bursts.
		client := &Client{hub: hub, conn: conn, send: make(chan []byte, 512), playerID: player.ID}
		hub.register <- client

		// Send welcome (full snapshot + self + server start time)
		snapshot := game.Snapshot()
		welcome := mustJSON(map[string]any{
			"type": "welcome",
			"payload": map[string]any{
				"self":         player,
				"currentState": snapshot,
				"serverStart":  game.StartTime.UTC().Format(time.RFC3339Nano),
			},
		})
		client.send <- welcome

		// Broadcast player_joined to others
		joined := mustJSON(map[string]any{
			"type":    "player_joined",
			"payload": player,
		})
		hub.Broadcast(joined)

		go client.writePump()
		go client.readPump()
	}
}

// Simple template cache to avoid reparsing every request

// (Old template cache removed for live-reload simplicity during development.)
// If needed later, implement mod-time aware caching.
type TemplateCache struct{ _ int }

var templateCache = &TemplateCache{}

// Example placeholder showing how mod-time reload could be added later.
func (_ *TemplateCache) Get(path string) (*template.Template, error) {
	return template.ParseFiles(path)
}

// Prevent unused import error in case future additions require time.
var _ = time.Now
