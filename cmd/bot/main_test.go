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
		name            string
		message         MessageCreator
		randFunc        func() float32
		allowedChannels map[string]bool
		shouldSend      bool
		expectedContent string
	}{
		{
			name:            "ランダム返信が無効(nil)の場合、何もされない",
			message:         &TestMessage{content: "any", channelID: "ch1", authorIsBot: false},
			randFunc:        func() float32 { return 0.1 },
			allowedChannels: nil,
			shouldSend:      false,
		},
		{
			name:            "許可されていないチャンネルでは何もされない",
			message:         &TestMessage{content: "any", channelID: "ch-other", authorIsBot: false},
			randFunc:        func() float32 { return 0.1 },
			allowedChannels: map[string]bool{"ch-allowed": true},
			shouldSend:      false,
		},
		{
			name:            "許可されたチャンネルで確率当選した場合、確率付きで返信する",
			message:         &TestMessage{content: "any", channelID: "ch-allowed", authorIsBot: false},
			randFunc:        func() float32 { return 0.15 },
			allowedChannels: map[string]bool{"ch-allowed": true},
			shouldSend:      true,
			expectedContent: "(1D100<=30) ＞ 16 ＞ 成功！\nテストセリフ1 | 第1幕：テストキャラ",
		},
		{
			name:            "許可されたチャンネルで確率落選した場合、何もされない",
			message:         &TestMessage{content: "any", channelID: "ch-allowed", authorIsBot: false},
			randFunc:        func() float32 { return 0.4 },
			allowedChannels: map[string]bool{"ch-allowed": true},
			shouldSend:      false,
		},
		{
			name:            "ボットの発言には(許可チャンネルでも)反応しない",
			message:         &TestMessage{content: "any", channelID: "ch-allowed", authorIsBot: true},
			randFunc:        func() float32 { return 0.1 },
			allowedChannels: map[string]bool{"ch-allowed": true},
			shouldSend:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sentContent = "" // 各テストの前にリセット
			messageCreate(mockSession, tc.message, testLines, tc.randFunc, tc.allowedChannels)

			if tc.shouldSend {
				if sentContent == "" {
					t.Error("メッセージが送信されませんでした")
				} else if sentContent != tc.expectedContent {
					t.Errorf("期待する内容と異なります。\ngot:  %q\nwant: %q", sentContent, tc.expectedContent)
				}
			} else {
				if sentContent != "" {
					t.Errorf("メッセージが送信されないはずが、送信されました: %s", sentContent)
				}
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
