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
	DefaultMaxFileSizeMB = 9.5 // discord's free tier limit is 10mb but will keep it at 9.5 or lower
)

func DownloadVideo(urlStr, messageID string) (string, error) {
	outputPath := filepath.Join(VideosDir, fmt.Sprintf("%s.mp4", messageID))
	size := fmt.Sprintf("%vM", GetMaxFileSizeMB())
	log.Printf("max file size: %v", size)

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
