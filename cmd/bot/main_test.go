package main

import (
	"path/filepath"
	"testing"
	"github.com/bwmarrin/discordgo"
)

// テスト用のセッション構造体
type TestSession struct {
	channelMessageSend func(string, string) error
}

func (s *TestSession) ChannelMessageSend(channelID, content string) error {
	return s.channelMessageSend(channelID, content)
}

// テスト用のCSVファイルパスを取得
func getTestCSVPath(filename string) string {
	// 一旦docker上での開発しか想定していないためpathはベタうち
	return filepath.Join("/app/test_data/", filename)
}

// loadLinesのテスト
func TestLoadLines(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		lines = []Line{}
		csvPath := getTestCSVPath("valid_serif-list.csv")
		if err := loadLines(csvPath); err != nil {
			t.Errorf("loadLines() failed: %v", err)
		}
		if len(lines) != 3 {
			t.Errorf("Expected 3 lines, got %d", len(lines))
		}
	})

	t.Run("FileNotFound", func(t *testing.T) {
		lines = []Line{}
		csvPath := "nonexistent.csv"
		if err := loadLines(csvPath); err == nil {
			t.Error("loadLines() should have failed for non-existent file")
		}
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		lines = []Line{}
		csvPath := getTestCSVPath("invalid_format_serif-list.csv")
		if err := loadLines(csvPath); err == nil {
			t.Error("loadLines() should have failed for invalid format")
		}
	})
}

// getRandomLineのテスト
func TestGetRandomLine(t *testing.T) {
	t.Run("WithLines", func(t *testing.T) {
		testLines := []Line{
			{Act: "第1幕", Character: "テストキャラ", Line: "テストセリフ1"},
			{Act: "第2幕", Character: "テストキャラ", Line: "テストセリフ2"},
			{Act: "第3幕", Character: "テストキャラ", Line: "テストセリフ3"},
		}
		response := getRandomLine(testLines)
		if response == "セリフが見つかりませんでした" {
			t.Error("getRandomLine() should return a line, not error message")
		}
	})

	t.Run("NoLines", func(t *testing.T) {
		response := getRandomLine([]Line{})
		if response != "セリフが見つかりませんでした" {
			t.Errorf("Expected error message, got %s", response)
		}
	})
}

// メッセージハンドラーのテスト
func TestMessageCreate(t *testing.T) {
	// モックセッションを作成
	mockSession := &TestSession{}

	t.Run("SerifCommand", func(t *testing.T) {
		var sentContent string
		var sentChannelID string
		mockSession.channelMessageSend = func(channelID string, content string) error {
			sentContent = content
			sentChannelID = channelID
			return nil
		}

		// テスト用のメッセージを作成
		testMessage := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				Content: "/serif",
				ChannelID: "test-channel",
			},
		}

		// テスト実行
		messageCreate(mockSession, &DiscordMessageCreateAdapter{testMessage}, []Line{
			{Act: "第1幕", Character: "テストキャラ", Line: "テストセリフ1"},
		})

		// メッセージが送信されたことを確認
		if sentContent == "" {
			t.Errorf("Expected message to be sent for /serif command")
		}

		// メッセージの内容が正しいことを確認
		expectedContent := "テストセリフ1 | 第1幕：テストキャラ"
		if sentContent != expectedContent {
			t.Errorf("Expected content to be %s, but got: %s", expectedContent, sentContent)
		}

		// チャンネルIDが正しいことを確認
		if sentChannelID != testMessage.ChannelID {
			t.Errorf("Expected channelID to be %s, but got: %s", testMessage.ChannelID, sentChannelID)
		}
	})

	t.Run("InvalidCommand", func(t *testing.T) {
		var sentContent string
		mockSession.channelMessageSend = func(channelID string, content string) error {
			sentContent = content
			return nil
		}

		// テスト用のメッセージを作成
		testMessage := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				Content: "invalid",
				ChannelID: "test-channel",
			},
		}

		// テスト実行
		messageCreate(mockSession, &DiscordMessageCreateAdapter{testMessage}, []Line{
			{Act: "第1幕", Character: "テストキャラ", Line: "テストセリフ1"},
		})

		// メッセージが送信されないことを確認
		if sentContent != "" {
			t.Errorf("Expected no message to be sent for invalid command, but got: %s", sentContent)
		}
	})
}
