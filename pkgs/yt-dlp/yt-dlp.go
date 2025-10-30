// Package ytdlp contains methods for dealing with yt-dlp
package ytdlp

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// SupportedDomains ensures we only want to restrict bot's usage to this domains
var SupportedDomains = []string{
	"www.instagram.com",
	"instagram.com",
	"x.com",
	"twitter.com",
	"www.tiktok.com",
	"tiktok.com",
	"www.youtube.com",
	"youtube.com",
}

// ErrMaxFilesizeExceeded is a custom error for exceeded file limit
var ErrMaxFilesizeExceeded = errors.New("video download aborted: file is larger than the maximum allowed size")

const (
	VideosDir            = "videos"
	DefaultMaxFileSizeMB = 10 // respect discord's free tier limit
)

func DownloadVideo(urlStr, messageID string) (string, error) {
	outputPath := filepath.Join(VideosDir, fmt.Sprintf("%s.mp4", messageID))
	size := fmt.Sprintf("%dM", GetMaxFileSizeMB())

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

func GetMaxFileSizeMB() int {
	maxSizeStr := os.Getenv("MAX_FILE_SIZE_MB")
	if maxSizeStr == "" {
		return DefaultMaxFileSizeMB
	}

	maxSize, err := strconv.Atoi(maxSizeStr)
	if err != nil {
		log.Printf("Invalid MAX_FILE_SIZE_MB '%s', using default %dMB", maxSizeStr, DefaultMaxFileSizeMB)
		return DefaultMaxFileSizeMB
	}

	log.Printf("Using max file size: %dMB", maxSize)
	return maxSize
}
