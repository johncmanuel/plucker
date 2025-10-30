package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// only want to restrict bot's usage to this domains
var supportedDomains = []string{
	"www.instagram.com",
	"instagram.com",
	"x.com",
	"twitter.com",
	"www.tiktok.com",
	"tiktok.com",
	"www.youtube.com",
	"youtube.com",
}

// custom error for exceeded file limit
var ErrMaxFilesizeExceeded = errors.New("video download aborted: file is larger than the maximum allowed size")

const (
	videosDir            = "videos"
	defaultMaxFileSizeMB = 10 // respect discord's free tier limit
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
		fmt.Println("error starting plucker")
	}

	dg.AddHandler(sendVideo)

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("Can't open connection")
	}

	// clean up directory
	err = RemoveContents(videosDir)
	if err != nil {
		log.Printf("Error removing %s: %v", videosDir, err)
	}

	err = os.MkdirAll(videosDir, 0o755)
	if err != nil {
		log.Fatalln("Failed to create videos directory:", err)
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

	// if content contains valid link(s) to some content, get the video(s) with yt-dlp
	urls := getUrls(m.Content)
	if len(urls) == 0 {
		return
	}

	for _, urlStr := range urls {
		if !isSupportedURL(urlStr) {
			continue
		}

		log.Printf("Found supported URL: %s in message %s", urlStr, m.ID)
		log.Printf("Downloading video from %s", urlStr)

		filePath, err := downloadVideo(urlStr, m.ID)
		if err != nil {
			log.Printf("Failed to download video: %v", err)
			s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("Sorry, I couldn't download this video, here's why: %v", err), m.Reference())
			err = RemoveContents(videosDir)
			if err != nil {
				log.Printf("Error removing %s: %v", videosDir, err)
			}
			continue
		}

		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("Failed to open downloaded file: %v", err)
			s.ChannelMessageSendReply(m.ChannelID, "Error: Could not open downloaded file.", m.Reference())
			err = RemoveContents(videosDir)
			if err != nil {
				log.Printf("Error removing %s: %v", videosDir, err)
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
		err = RemoveContents(videosDir)
		if err != nil {
			log.Printf("Error removing %s: %v", videosDir, err)
		}
		log.Printf("Cleared contents of %s", videosDir)
	}
}

func getUrls(text string) []string {
	// only accept https links!
	regex := `(https):\/\/([\w_-]+(?:(?:\.[\w_-]+)+))([\w.,@?^=%&:\/~+#-]*[\w@?^=%&\/~+#-])`
	re, err := regexp.Compile(regex)
	if err != nil {
		// this'll only happen if the regex above is written badly
		fmt.Println("Error compiling regex", regex)
		return nil
	}

	return re.FindAllString(text, -1)
}

func isSupportedURL(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		log.Printf("Failed to parse URL: %s", urlStr)
		return false
	}

	return slices.Contains(supportedDomains, parsedURL.Host)
}

func downloadVideo(urlStr, messageID string) (string, error) {
	outputPath := filepath.Join(videosDir, fmt.Sprintf("%s.mp4", messageID))
	size := fmt.Sprintf("%dM", getMaxFileSizeMB())

	// Args explanation
	// --max-filesize: aborts downloads if download exceeds a specified limit
	// --merge-output-format mp4: Ensures the final file is an mp4.
	// -o: Specifies the output path.
	cmd := exec.Command("yt-dlp",
		"--max-filesize", size,
		"--merge-output-format", "mp4",
		"-o", outputPath,
		urlStr,
	)

	output, err := cmd.CombinedOutput()

	// yt-dlp doesn't treat download aborts as errors, so look through its output for this specific
	// error
	if strings.Contains(string(output), "File is larger than max-filesize") {
		log.Printf("yt-dlp aborted for %s: file larger than max file size: %s", urlStr, size)

		return "", ErrMaxFilesizeExceeded
	}

	if err != nil {
		log.Printf("yt-dlp error for URL %s: %s\nOutput: %s", urlStr, err, string(output))
		return "", fmt.Errorf("yt-dlp failed: %s", err)
	}

	log.Printf("Successfully downloaded video, path located at: %s", outputPath)
	return outputPath, nil
}

func getMaxFileSizeMB() int {
	maxSizeStr := os.Getenv("MAX_FILE_SIZE_MB")
	if maxSizeStr == "" {
		return defaultMaxFileSizeMB
	}

	maxSize, err := strconv.Atoi(maxSizeStr)
	if err != nil {
		log.Printf("Invalid MAX_FILE_SIZE_MB '%s', using default %dMB", maxSizeStr, defaultMaxFileSizeMB)
		return defaultMaxFileSizeMB
	}

	log.Printf("Using max file size: %dMB", maxSize)
	return maxSize
}

func RemoveContents(dir string) error {
	// remove contents of directory without removing directory itself
	// https://stackoverflow.com/a/33451503
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
