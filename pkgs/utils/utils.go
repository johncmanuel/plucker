// Package utils contains helper methods and other useful stuff
package utils

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	ytdlp "github.com/johncmanuel/plucker/pkgs/yt-dlp"
)

// GetUrls finds all https URLs in a given string of text.
func GetUrls(text string) []string {
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

// IsSupportedURL checks if a given URL string is from a supported domain.
func IsSupportedURL(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		log.Printf("Failed to parse URL: %s", urlStr)
		return false
	}

	return slices.Contains(ytdlp.SupportedDomains, parsedURL.Host)
}

// RemoveContents removes all files and subdirectories from a directory
// without removing the directory itself.
func RemoveContents(dir string) error {
	// https://stackoverflow.com/a/33451503
	d, err := os.Open(dir)
	if err != nil {
		// If directory doesn't exist, that's fine
		if os.IsNotExist(err) {
			return nil
		}
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
