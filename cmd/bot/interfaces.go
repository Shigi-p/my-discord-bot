package main

import "github.com/bwmarrin/discordgo"

// メッセージ送信インターフェース
type ChannelMessageSender interface {
	ChannelMessageSend(string, string) error
}

// メッセージ作成インターフェース
type MessageCreator interface {
	Content() string
	ChannelID() string
	AuthorIsBot() bool
}

// インタラクション応答インターフェース
type InteractionResponder interface {
	InteractionRespond(*discordgo.Interaction, *discordgo.InteractionResponse) error
}
