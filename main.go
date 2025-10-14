package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func main() {
	token := ""
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error starting plucker")
	}

	dg.AddHandler(sendVideo)
}

func sendVideo(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// if content contains valid link to some content, get the video with yt-dlp
}
