package main

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	// TODO: set up token retrieval from .env file or equivalent
	token := ""
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error starting plucker")
	}

	dg.AddHandler(sendVideo)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("Can't open connection")
	}

	fmt.Println("Plucker is now running! Use Ctrl+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("Shutting down Plucker.")
	dg.Close()
}

func sendVideo(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// if content contains valid link(s) to some content, get the video with yt-dlp
	urls := getUrls(m.Content)
	if len(urls) == 0 {
		return
	}
}

func getUrls(text string) []string {
	// only accept https links!
	regex := `(https):\/\/([\w_-]+(?:(?:\.[\w_-]+)+))([\w.,@?^=%&:\/~+#-]*[\w@?^=%&\/~+#-])`
	re, err := regexp.Compile(regex)
	if err != nil {
		fmt.Println("Error compiling regex", regex)
		return nil
	}

	return re.FindAllString(text, -1)
}
