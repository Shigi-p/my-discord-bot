package main

import (
	"path/filepath"
	"testing"
)

// テスト用のセッション構造体
type TestSession struct {
	channelMessageSend func(string, string) error
}

func (s *TestSession) ChannelMessageSend(channelID, content string) error {
	return s.channelMessageSend(channelID, content)
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
			name:          "/serifコマンドには確率に関わらず必ず返信する",
			message:       &TestMessage{content: "/serif", channelID: "ch2", authorIsBot: false},
			randFunc:      func() float32 { return 0.9 }, // 確率的には落選するがコマンドなので送信されるはず
			shouldSend:    true,
			expectedError: "/serifコマンドには返信するはずが、メッセージが送信されませんでした",
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
