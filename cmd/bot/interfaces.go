package main

// メッセージ送信インターフェース
type ChannelMessageSender interface {
	ChannelMessageSend(string, string) error
}

// メッセージ作成インターフェース
type MessageCreator interface {
	Content() string
	ChannelID() string
}
