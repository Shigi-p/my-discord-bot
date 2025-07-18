package main

import (
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// テスト用のセッション構造体
type TestSession struct {
	channelMessageSend   func(string, string) error
	interactionRespond func(*discordgo.Interaction, *discordgo.InteractionResponse) error
}

func (s *TestSession) ChannelMessageSend(channelID, content string) error {
	return s.channelMessageSend(channelID, content)
}

func (s *TestSession) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse) error {
	return s.interactionRespond(i, r)
}

// テスト用のメッセージ構造体
type TestMessage struct {
	content     string
	channelID   string
	authorIsBot bool
}

func (m *TestMessage) Content() string {
	return m.content
}
func (m *TestMessage) ChannelID() string {
	return m.channelID
}
func (m *TestMessage) AuthorIsBot() bool {
	return m.authorIsBot
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
	testLines := []Line{{Act: "第1幕", Character: "テストキャラ", Line: "テストセリフ1"}}
	var sentContent string
	mockSession := &TestSession{
		channelMessageSend: func(channelID, content string) error {
			sentContent = content
			return nil
		},
	}

	// テストケースを構造体のスライスとして定義
	testCases := []struct {
		name          string
		message       MessageCreator
		randFunc      func() float32
		shouldSend    bool
		expectedError string
	}{
		{
			name:          "ボットの発言には反応しない",
			message:       &TestMessage{content: "/serif", channelID: "ch1", authorIsBot: true},
			randFunc:      func() float32 { return 0.1 }, // 確率的には当選するがボットなので無視されるはず
			shouldSend:    false,
			expectedError: "ボットの発言には返信しないはずですが、メッセージが送信されました",
		},
		{
			name:          "通常メッセージに30%の確率で返信する",
			message:       &TestMessage{content: "こんにちは", channelID: "ch3", authorIsBot: false},
			randFunc:      func() float32 { return 0.29 }, // 30%未満なので当選
			shouldSend:    true,
			expectedError: "30%の確率で返信するはずが、メッセージが送信されませんでした",
		},
		{
			name:          "通常メッセージに70%の確率で返信しない",
			message:       &TestMessage{content: "こんばんは", channelID: "ch4", authorIsBot: false},
			randFunc:      func() float32 { return 0.3 }, // 30%以上なので落選
			shouldSend:    false,
			expectedError: "70%の確率で返信しないはずが、メッセージが送信されました",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sentContent = "" // 各テストの前にリセット

			messageCreate(mockSession, tc.message, testLines, tc.randFunc)

			if tc.shouldSend && sentContent == "" {
				t.Error(tc.expectedError)
			}

			if !tc.shouldSend && sentContent != "" {
				t.Errorf("%s: %s", tc.expectedError, sentContent)
			}
		})
	}
}

// スラッシュコマンドハンドラーのテスト
func TestInteractionCreate(t *testing.T) {
	testLines := []Line{{Act: "第1幕", Character: "テストキャラ", Line: "テストセリフ1"}}
	var capturedResponse *discordgo.InteractionResponse
	mockSession := &TestSession{
		interactionRespond: func(i *discordgo.Interaction, r *discordgo.InteractionResponse) error {
			capturedResponse = r
			return nil
		},
	}

	t.Run("serifコマンドが実行された場合、応答が返される", func(t *testing.T) {
		capturedResponse = nil // reset

		testInteraction := &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "serif",
				},
			},
		}

		interactionCreate(mockSession, testInteraction, testLines)

		if capturedResponse == nil {
			t.Fatal("InteractionRespondが呼び出されませんでした")
		}
		if capturedResponse.Type != discordgo.InteractionResponseChannelMessageWithSource {
			t.Errorf("期待する応答タイプと異なります. got=%v, want=%v", capturedResponse.Type, discordgo.InteractionResponseChannelMessageWithSource)
		}
		expectedContent := "テストセリフ1 | 第1幕：テストキャラ"
		if capturedResponse.Data.Content != expectedContent {
			t.Errorf("期待する応答内容と異なります. got=%q, want=%q", capturedResponse.Data.Content, expectedContent)
		}
	})

	t.Run("serif以外のコマンドが実行された場合、何もされない", func(t *testing.T) {
		capturedResponse = nil // reset

		testInteraction := &discordgo.InteractionCreate{
			Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Name: "other-command",
				},
			},
		}

		interactionCreate(mockSession, testInteraction, testLines)

		if capturedResponse != nil {
			t.Error("serif以外のコマンドには応答しないはずですが、応答が返されました")
		}
	})
}
