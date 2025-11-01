package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"

	"github.com/johncmanuel/plucker/pkgs/utils"
	ytdlp "github.com/johncmanuel/plucker/pkgs/yt-dlp"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, proceeding with environment variables")
	}

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatalln("Error: BOT_TOKEN environment variable not set.")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalln("error starting plucker")
	}

	dg.AddHandler(sendVideo)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		log.Fatalln("Can't open connection")
	}

	// clean up directory
	err = utils.RemoveContents(ytdlp.VideosDir)
	if err != nil {
		log.Printf("Error removing contents of %s: %v", ytdlp.VideosDir, err)
	}

	err = os.MkdirAll(ytdlp.VideosDir, 0o755)
	if err != nil {
		log.Fatalln("Failed to create videos directory:", err)
	}

	fmt.Println("Plucker is now running! Use Ctrl+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("Shutting down Plucker.")

	err = dg.Close()
	if err != nil {
		log.Printf("could not close gracefully: %s", err)
	}
}

func sendVideo(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// if content contains valid link(s) to some content, get the video(s) with yt-dlp
	urls := utils.GetUrls(m.Content)
	if len(urls) == 0 {
		return
	}

	for _, urlStr := range urls {
		if !utils.IsSupportedURL(urlStr) {
			continue
		}

		log.Printf("Found supported URL: %s in message %s", urlStr, m.ID)
		log.Printf("Downloading video from %s", urlStr)

		filePath, err := ytdlp.DownloadVideo(urlStr, m.ID)
		if err != nil {
			log.Printf("Failed to download video: %v", err)
			s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Error: unable to download video. %v", err), m.Reference())

			err = utils.RemoveContents(ytdlp.VideosDir)
			if err != nil {
				log.Printf("Error removing %s: %v", ytdlp.VideosDir, err)
			}
			continue
		}

		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("Failed to open downloaded file: %v", err)
			s.ChannelMessageSendReply(m.ChannelID, "Error: Could not open downloaded file.", m.Reference())

			err = utils.RemoveContents(ytdlp.VideosDir)
			if err != nil {
				log.Printf("Error removing %s: %v", ytdlp.VideosDir, err)
			}
			continue
		}

		fileInfo, err := file.Stat()
		if err != nil {
			log.Printf("Error getting file info: %v\n", err)
			s.ChannelMessageSendReply(m.ChannelID, "Error: could not get file info for one of them.", m.Reference())

			file.Close()
			err = utils.RemoveContents(ytdlp.VideosDir)
			if err != nil {
				log.Printf("Error removing %s: %v", ytdlp.VideosDir, err)
			}
			continue
		}

		fileSizeMb := float64(fileInfo.Size()) / (1024 * 1024)
		log.Printf("File size: %.2f MB\n", fileSizeMb)

		if fileSizeMb > float64(ytdlp.DefaultMaxFileSizeMB) {
			log.Printf("file size, %.2f, is larger than default max file size, can't send it to discord!", fileSizeMb)
			s.ChannelMessageSendReply(m.ChannelID, "Error: video is larger than discord's file limit, can't send it!", m.Reference())

			file.Close()
			err = utils.RemoveContents(ytdlp.VideosDir)
			if err != nil {
				log.Printf("Error removing %s: %v", ytdlp.VideosDir, err)
			}
			continue
		}

		_, err = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Files: []*discordgo.File{
				{
					Name:   filepath.Base(filePath),
					Reader: file,
				},
			},
			Reference: m.Reference(),
		})
		if err != nil {
			log.Printf("Failed to send file to Discord: %v", err)
			s.ChannelMessageSendReply(m.ChannelID, "Error: Could not send file to Discord.", m.Reference())
		}

		file.Close()
		err = utils.RemoveContents(ytdlp.VideosDir)
		if err != nil {
			log.Printf("Error removing %s: %v", ytdlp.VideosDir, err)
		}
		log.Printf("Cleared contents of %s", ytdlp.VideosDir)
	}
}
