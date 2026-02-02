package handlers

import (
	"regexp"
	"strings"
)

// ExtractYouTubeVideoID extracts YouTube video ID from various URL formats
// Supports:
// - https://www.youtube.com/watch?v=VIDEO_ID
// - https://youtu.be/VIDEO_ID
// - https://www.youtube.com/embed/VIDEO_ID
// - <iframe src="https://www.youtube.com/embed/VIDEO_ID"...>
func ExtractYouTubeVideoID(input string) string {
	if input == "" {
		return ""
	}

	// Clean input
	input = strings.TrimSpace(input)

	// Pattern 1: Watch URL with query params
	// https://www.youtube.com/watch?v=VIDEO_ID
	re := regexp.MustCompile(`(?:youtube\.com\/watch\?v=|youtu\.be\/)([a-zA-Z0-9_-]{11})`)
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}

	// Pattern 2: Embed URL or iframe
	// https://www.youtube.com/embed/VIDEO_ID
	re = regexp.MustCompile(`youtube\.com\/embed\/([a-zA-Z0-9_-]{11})`)
	matches = re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}

	// Pattern 3: Short URL
	// https://youtu.be/VIDEO_ID
	re = regexp.MustCompile(`youtu\.be\/([a-zA-Z0-9_-]{11})`)
	matches = re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return matches[1]
	}

	// If no match, return empty
	return ""
}

// GetYouTubeEmbedURL converts various YouTube URL formats to standard embed URL
func GetYouTubeEmbedURL(input string) string {
	videoID := ExtractYouTubeVideoID(input)
	if videoID == "" {
		return ""
	}
	return "https://www.youtube.com/embed/" + videoID
}
