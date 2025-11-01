// Package ytdlp contains methods for dealing with yt-dlp
package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

// ErrDownloadTimeout is a custom error for timeouts. Mainly used for manually stopping users from
// downloading large videos
var ErrDownloadTimeout = errors.New("video download aborted: process took too long")

var ErrProcessKilled = errors.New("video download failed: process was killed (likely out of memory)")

const (
	VideosDir                     = "videos"
	DefaultMaxFileSizeMB          = 10
	defaultDownloadTimeoutSeconds = 15 // This seems like a good default threshold
)

func DownloadVideo(urlStr, messageID string) (string, error) {
	outputPath := filepath.Join(VideosDir, fmt.Sprintf("%s.mp4", messageID))
	size := fmt.Sprintf("%vM", GetMaxFileSizeMB())
	log.Printf("max file size: %v", size)

	timeout := getDownloadTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// yt-dlp args explanation
	// --max-filesize: aborts downloads if download exceeds a specified limit
	// --merge-output-format mp4: Ensures the final file is an mp4.
	// -o: Specifies the output path.
	//
	// TODO: --max-filesize doesn't tend to work with links with DASH or HLS downloads
	// as seen in X.com links, which uses HLS
	// https://github.com/yt-dlp/yt-dlp/issues/10663#issuecomment-2271765154
	// https://github.com/ytdl-org/youtube-dl/issues/22133#issuecomment-2388020660
	// Test output:
	// ...
	// [twitter] Extracting URL: https://x.com/FGC_Daily/status/1983270990479262007
	// [twitter] 1983270990479262007: Downloading guest token
	// [twitter] 1983270990479262007: Downloading GraphQL JSON
	// [twitter] 1983270990479262007: Downloading m3u8 information
	// [info] 1983270399568719872: Downloading 1 format(s): hls-2025+hls-audio-128000-Audio
	// [hlsnative] Downloading m3u8 manifest
	// ...

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--max-filesize", size,
		"--merge-output-format", "mp4",
		"-o", outputPath,
		urlStr,
	)

	doneChan := make(chan error, 1)
	outputChan := make(chan []byte, 1)

	go func() {
		output, err := cmd.CombinedOutput()
		outputChan <- output
		doneChan <- err
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	countdown := int(timeout.Seconds())
	log.Printf("Starting download for %s (timeout: %ds)", urlStr, countdown)

	for {
		select {
		case <-ctx.Done():
			<-doneChan
			output := <-outputChan
			log.Printf("yt-dlp process timed out for URL %s (limit: %v)\nOutput: %s", urlStr, timeout, string(output))
			return "", ErrDownloadTimeout

		case err := <-doneChan:
			output := <-outputChan
			if strings.Contains(string(output), "File is larger than max-filesize") {
				log.Printf("yt-dlp aborted for %s: file larger than max file size: %s", urlStr, size)

				return "", ErrMaxFilesizeExceeded
			}

			if err != nil {
				// apparently this can happen, so handle this appropiately
				if err.Error() == "signal: killed" {
					log.Printf("yt-dlp process was killed (likely OOM) for URL %s\nOutput: %s", urlStr, string(output))
					return "", ErrProcessKilled
				}

				// handle any other error from yt-dlp or related
				log.Printf("yt-dlp error for URL %s: %s\nOutput: %s", urlStr, err, string(output))
				return "", fmt.Errorf("yt-dlp failed: %s", err)
			}

			log.Printf("Successfully downloaded video, path located at: %s", outputPath)
			return outputPath, nil

		case <-ticker.C:
			countdown--
			// log every 10 seconds or every second for the last 5 seconds
			if countdown <= 5 || countdown%10 == 0 {
				log.Printf("Downloading %s... (time remaining: %ds)", urlStr, countdown)
			}
		}
	}
}

func GetMaxFileSizeMB() float32 {
	maxSizeStr := os.Getenv("MAX_FILE_SIZE_MB")
	if maxSizeStr == "" {
		return DefaultMaxFileSizeMB
	}

	maxSize, err := strconv.Atoi(maxSizeStr)
	if err != nil {
		log.Printf("Invalid MAX_FILE_SIZE_MB '%s', using default %vMB", maxSizeStr, DefaultMaxFileSizeMB)
		return DefaultMaxFileSizeMB
	}

	log.Printf("Using max file size: %dMB", maxSize)
	return float32(maxSize)
}

func getDownloadTimeout() time.Duration {
	timeoutStr := os.Getenv("DOWNLOAD_TIMEOUT_SECONDS")
	if timeoutStr == "" {
		return defaultDownloadTimeoutSeconds * time.Second
	}

	timeoutSec, err := strconv.Atoi(timeoutStr)
	if err != nil {
		log.Printf("Invalid DOWNLOAD_TIMEOUT_SECONDS '%s', using default %ds", timeoutStr, defaultDownloadTimeoutSeconds)
		return defaultDownloadTimeoutSeconds * time.Second
	}

	log.Printf("Using download timeout: %ds", timeoutSec)
	return time.Duration(timeoutSec) * time.Second
}
