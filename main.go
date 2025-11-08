package main

import (
	"log"
	"net/http"
)

func main() {
	game := NewGame()
	hub := NewHub(game)
	game.Hub = hub

	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/", HandleRoot(game))
	mux.Handle("/static/", HandleStatic())
	mux.HandleFunc("/ws", HandleWebSocket(hub, game))

	addr := ":8080"
	log.Printf("Booper server starting on http://localhost%v", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
