package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		fmt.Println("BOT_TOKEN is not set. Plz set it in .env file.")
		return
	}

	// Create new discord session (know I am online to discord server)
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating session:", err)
		return
	}

	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening session:", err)
		return
	}

	fmt.Println("Bot started. Press CTRL+C to shutdown.")

	// Wait for program to exit
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Close session
	fmt.Println("Bot to shutdown.")
	dg.Close()
}


// test function
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "!ping" {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}
}