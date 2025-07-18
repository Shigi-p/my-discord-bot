package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "serif",
			Description: "ランダムなセリフを返します",
		},
	}
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

func (a *DiscordMessageCreateAdapter) AuthorIsBot() bool {
	return a.MessageCreate.Author.Bot
}

type DiscordSessionAdapter struct {
	*discordgo.Session
}

func (a *DiscordSessionAdapter) ChannelMessageSend(channelID, content string) error {
	_, err := a.Session.ChannelMessageSend(channelID, content)
	return err
}

func (a *DiscordSessionAdapter) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse) error {
	return a.Session.InteractionRespond(i, r)
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
		messageCreate(&DiscordSessionAdapter{s}, &DiscordMessageCreateAdapter{m}, lines, rand.Float32)
	}
	// スラッシュコマンド用のハンドラーを追加
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		interactionCreate(&DiscordSessionAdapter{s}, i, lines)
	})

	dg.AddHandler(wrappedHandler)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening session:", err)
		return
	}

	fmt.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", v)
		if err != nil {
			log.Fatalf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	fmt.Println("Bot started. Press CTRL+C to shutdown.")

	// Wait for program to exit
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("Removing commands...")
	for _, v := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, "", v.ID)
		log.Printf("Cannot delete '%v' command: %v", v.Name, err)
	}

	// Close session
	fmt.Println("Bot to shutdown.")
	dg.Close()
}

// sendRandomLine は、与えられたセリフリストからランダムに一つを選んで送信する共通関数
func sendRandomLine(s ChannelMessageSender, channelID string, lines []Line) {
	response := getRandomLine(lines)
	s.ChannelMessageSend(channelID, response)
}

func getRandomLine(lines []Line) string {
	if len(lines) == 0 {
		return "セリフが見つかりませんでした"
	}
	index := rand.Intn(len(lines))
	line := lines[index]
	return fmt.Sprintf("%s | %s：%s", line.Line, line.Act, line.Character) // TODO: フォーマットを修正
}

// メッセージハンドラー
func messageCreate(s ChannelMessageSender, m MessageCreator, lines []Line, randFunc func() float32) {
	// ボット自身の発言には反応しない
	if m.AuthorIsBot() {
		return
	}

	shouldReplyRandomly := randFunc() < 0.3

	// 30%の確率で返信する
	if shouldReplyRandomly {
		sendRandomLine(s, m.ChannelID(), lines)
	}
}

// スラッシュコマンドハンドラー
func interactionCreate(s InteractionResponder, i *discordgo.InteractionCreate, lines []Line) {
	if i.Interaction.Type == discordgo.InteractionApplicationCommand {
		if i.Interaction.ApplicationCommandData().Name == "serif" {
			responseContent := getRandomLine(lines)
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: responseContent,
				},
			})
			if err != nil {
				log.Printf("Failed to respond to interaction: %v", err)
			}
		}
	}
}
