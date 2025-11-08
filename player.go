package main

import (
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Player struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func NewPlayer() *Player {
	return &Player{
		ID:   uuid.New().String(),
		Name: randomName(),
	}
}

func randomName() string {
	rand.Seed(time.Now().UnixNano())
	adjectives := []string{"Silly", "Brave", "Sneaky", "Bouncy", "Zippy", "Chill", "Witty", "Cosmic", "Sunny", "Spicy"}
	animals := []string{"Wombat", "Otter", "Panda", "Llama", "Gecko", "Dolphin", "Badger", "Kiwi", "Fox", "Capybara"}
	adj := adjectives[rand.Intn(len(adjectives))]
	ani := animals[rand.Intn(len(animals))]
	return strings.TrimSpace(adj + " " + ani)
}
