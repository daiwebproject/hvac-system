package handlers

import (
	"testing"
)

func TestExtractYouTubeVideoID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Watch URL
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://youtube.com/watch?v=dQw4w9WgXcQ&feature=share", "dQw4w9WgXcQ"},

		// Short URL
		{"https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ"},

		// Embed URL
		{"https://www.youtube.com/embed/dQw4w9WgXcQ", "dQw4w9WgXcQ"},

		// Iframe
		{`<iframe width="560" height="315" src="https://www.youtube.com/embed/dQw4w9WgXcQ" frameborder="0"></iframe>`, "dQw4w9WgXcQ"},

		// Invalid
		{"https://example.com", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := ExtractYouTubeVideoID(tt.input)
		if result != tt.expected {
			t.Errorf("ExtractYouTubeVideoID(%q) = %q; want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetYouTubeEmbedURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "https://www.youtube.com/embed/dQw4w9WgXcQ"},
		{"https://youtu.be/dQw4w9WgXcQ", "https://www.youtube.com/embed/dQw4w9WgXcQ"},
		{"https://example.com", ""},
	}

	for _, tt := range tests {
		result := GetYouTubeEmbedURL(tt.input)
		if result != tt.expected {
			t.Errorf("GetYouTubeEmbedURL(%q) = %q; want %q", tt.input, result, tt.expected)
		}
	}
}
