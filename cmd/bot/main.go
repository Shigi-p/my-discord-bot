package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// データ構造体
type Line struct {
	Act      string
	Character string
	Line     string
}

var lines []Line

type DiscordMessageCreateAdapter struct {
	*discordgo.MessageCreate
}

func (a *DiscordMessageCreateAdapter) Content() string {
	return a.MessageCreate.Content
}

func (a *DiscordMessageCreateAdapter) ChannelID() string {
	return a.MessageCreate.ChannelID
}

type DiscordSessionAdapter struct {
	*discordgo.Session
}

func (a *DiscordSessionAdapter) ChannelMessageSend(channelID, content string) error {
	_, err := a.Session.ChannelMessageSend(channelID, content)
	return err
}

func loadLines(csvPath string) error {
	lines = []Line{}

	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("failed to open csv file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read csv: %w", err)
	}

	// Skip header
	for _, record := range records[1:] {
		if len(record) != 3 {
			continue
		}
		lines = append(lines, Line{
			Act:      record[0],
			Character: record[1],
			Line:     record[2],
		})
	}

	return nil
}

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		fmt.Println("BOT_TOKEN is not set. Plz set it in .env file.")
		return
	}

	baseDir := filepath.Dir(os.Args[0])
	csvPath := filepath.Join(baseDir, "../serif-list.csv")
	if err := loadLines(csvPath); err != nil {
		fmt.Printf("Error loading lines: %v\n", err)
		return
	}

	rand.Seed(time.Now().UnixNano())

	// Create new discord session (know I am online to discord server)
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating session:", err)
		return
	}

	// ハンドラーをラップ
	wrappedHandler := func(s *discordgo.Session, m *discordgo.MessageCreate) {
		messageCreate(&DiscordSessionAdapter{s}, &DiscordMessageCreateAdapter{m}, lines)
	}
	dg.AddHandler(wrappedHandler)
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

func getRandomLine(lines []Line) string {
	if len(lines) == 0 {
		return "セリフが見つかりませんでした"
	}
	index := rand.Intn(len(lines))
	line := lines[index]
	return fmt.Sprintf("%s | %s：%s", line.Line, line.Act, line.Character)
}

// メッセージハンドラー
func messageCreate(s ChannelMessageSender, m MessageCreator, lines []Line) {
	if strings.HasPrefix(strings.ToLower(m.Content()), "/serif") {
		response := getRandomLine(lines)
		s.ChannelMessageSend(m.ChannelID(), response)
	}
}
