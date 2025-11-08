package main

import (
	"encoding/json"
	"sync"
	"time"
)

type GameState struct {
	Players       map[string]*Player         `json:"players"`
	BoopLog       map[string]map[string]bool `json:"boopLog"`
	BoopsMade     map[string]int             `json:"boopsMade"`
	BoopsReceived map[string]int             `json:"boopsReceived"`
	mu            sync.Mutex
}

type Game struct {
	State     *GameState
	Hub       *Hub
	StartTime time.Time
}

func NewGame() *Game {
	return &Game{
		State: &GameState{
			Players:       make(map[string]*Player),
			BoopLog:       make(map[string]map[string]bool),
			BoopsMade:     make(map[string]int),
			BoopsReceived: make(map[string]int),
		},
		StartTime: time.Now(),
	}
}

func (g *Game) AddPlayer(p *Player) {
	g.State.mu.Lock()
	defer g.State.mu.Unlock()
	g.State.Players[p.ID] = p
}

func (g *Game) RemovePlayer(id string) {
	g.State.mu.Lock()
	defer g.State.mu.Unlock()
	delete(g.State.Players, id)
	// Optional: clean up boop log entries
	delete(g.State.BoopLog, id)
	for _, inner := range g.State.BoopLog {
		delete(inner, id)
	}
	delete(g.State.BoopsMade, id)
	delete(g.State.BoopsReceived, id)
}

// RecordBoop registers a boop such that only the latest direction between two players is stored.
// Example: A boops B -> store A->B. B boops A -> remove A->B, store B->A. Glow follows latest booper.
// RecordBoop attempts to register a boop.
// Returns true if the boop changed state (i.e., not a duplicate from the same booper to the same target).
func (g *Game) RecordBoop(booperID, boopedID string) bool {
	g.State.mu.Lock()
	defer g.State.mu.Unlock()
	// If this exact boop already exists, reject duplicate.
	if g.State.BoopLog[booperID] != nil && g.State.BoopLog[booperID][boopedID] {
		return false
	}
	// Remove opposite direction if present so only one direction remains.
	if g.State.BoopLog[boopedID] != nil {
		delete(g.State.BoopLog[boopedID], booperID)
		if len(g.State.BoopLog[boopedID]) == 0 {
			delete(g.State.BoopLog, boopedID)
		}
	}
	if g.State.BoopLog[booperID] == nil {
		g.State.BoopLog[booperID] = make(map[string]bool)
	}
	g.State.BoopLog[booperID][boopedID] = true
	// Update counters
	g.State.BoopsMade[booperID] = g.State.BoopsMade[booperID] + 1
	g.State.BoopsReceived[boopedID] = g.State.BoopsReceived[boopedID] + 1
	return true
}

// Snapshot returns a deep-copied JSON-friendly snapshot of state
func (g *Game) Snapshot() map[string]any {
	g.State.mu.Lock()
	defer g.State.mu.Unlock()

	players := make(map[string]*Player, len(g.State.Players))
	for id, p := range g.State.Players {
		players[id] = &Player{ID: p.ID, Name: p.Name}
	}
	boopLog := make(map[string]map[string]bool, len(g.State.BoopLog))
	for a, inner := range g.State.BoopLog {
		boopLog[a] = make(map[string]bool, len(inner))
		for b, v := range inner {
			boopLog[a][b] = v
		}
	}
	boopsMade := make(map[string]int, len(g.State.BoopsMade))
	for id, c := range g.State.BoopsMade {
		boopsMade[id] = c
	}
	boopsReceived := make(map[string]int, len(g.State.BoopsReceived))
	for id, c := range g.State.BoopsReceived {
		boopsReceived[id] = c
	}
	return map[string]any{
		"players":       players,
		"boopLog":       boopLog,
		"boopsMade":     boopsMade,
		"boopsReceived": boopsReceived,
	}
}

// Envelope is the base message wrapper
// Using explicit struct keeps protocol tidy and discoverable
// Payload stays as raw json in receive path and concrete types when sending

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
