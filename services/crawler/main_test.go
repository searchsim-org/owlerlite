package main

import (
	"testing"
)

// TestHammingDistance verifies the Hamming distance calculation
func TestHammingDistance(t *testing.T) {
	tests := []struct {
		name     string
		h1       uint64
		h2       uint64
		expected int
	}{
		{"identical", 0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF, 0},
		{"one bit different", 0x0000000000000001, 0x0000000000000000, 1},
		{"two bits different", 0x0000000000000003, 0x0000000000000000, 2},
		{"max difference", 0xFFFFFFFFFFFFFFFF, 0x0000000000000000, 64},
		{"typical difference", 0xF0F0F0F0F0F0F0F0, 0x0F0F0F0F0F0F0F0F, 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hammingDistance(tt.h1, tt.h2)
			if got != tt.expected {
				t.Errorf("hammingDistance(%x, %x) = %d, want %d", tt.h1, tt.h2, got, tt.expected)
			}
		})
	}
}

// TestThresholdClassification tests the two-threshold detection logic
func TestThresholdClassification(t *testing.T) {
	// τ₁ = 0.97 → Hamming ≤ 2 (unchanged)
	// τ₂ = 0.90 → Hamming ≥ 6 (definitely changed)
	// Intermediate: 3-5 (needs embedding check)

	tests := []struct {
		name           string
		hammingDist    int
		expectedAction string
	}{
		{"unchanged (dist=0)", 0, "skip"},
		{"unchanged (dist=1)", 1, "skip"},
		{"unchanged (dist=2)", 2, "skip"},
		{"intermediate (dist=3)", 3, "embedding_check"},
		{"intermediate (dist=4)", 4, "embedding_check"},
		{"intermediate (dist=5)", 5, "embedding_check"},
		{"changed (dist=6)", 6, "reindex"},
		{"changed (dist=10)", 10, "reindex"},
		{"changed (dist=64)", 64, "reindex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var action string
			if tt.hammingDist <= TAU1_HAMMING {
				action = "skip"
			} else if tt.hammingDist >= TAU2_HAMMING {
				action = "reindex"
			} else {
				action = "embedding_check"
			}

			if action != tt.expectedAction {
				t.Errorf("hammingDist=%d → action=%s, want %s", tt.hammingDist, action, tt.expectedAction)
			}
		})
	}
}

// TestCosineSimilarity verifies the cosine similarity calculation
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
		epsilon  float64
	}{
		{"identical vectors", []float64{1, 0, 0}, []float64{1, 0, 0}, 1.0, 0.001},
		{"orthogonal vectors", []float64{1, 0, 0}, []float64{0, 1, 0}, 0.0, 0.001},
		{"opposite vectors", []float64{1, 0, 0}, []float64{-1, 0, 0}, -1.0, 0.001},
		{"similar vectors", []float64{1, 1, 0}, []float64{1, 0.9, 0}, 0.99, 0.02},
		{"empty vectors", []float64{}, []float64{}, 0.0, 0.001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if got < tt.expected-tt.epsilon || got > tt.expected+tt.epsilon {
				t.Errorf("cosineSimilarity(%v, %v) = %f, want %f (±%f)", tt.a, tt.b, got, tt.expected, tt.epsilon)
			}
		})
	}
}

// TestSplitByHeadings tests the text chunking logic
func TestSplitByHeadings(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLen    int
		minChunks int
		maxChunks int
	}{
		{"empty text", "", 100, 1, 1},
		{"short text", "Hello world", 100, 1, 1},
		{"heading splits", "# Title\nContent\n# Another\nMore content", 100, 2, 3},
		{"length splits", "Word " + string(make([]byte, 500)), 100, 2, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := splitByHeadings(tt.text, tt.maxLen)
			if len(chunks) < tt.minChunks || len(chunks) > tt.maxChunks {
				t.Errorf("splitByHeadings produced %d chunks, want %d-%d", len(chunks), tt.minChunks, tt.maxChunks)
			}
		})
	}
}

// TestIsDisallowed tests robots.txt parsing
func TestIsDisallowed(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow: /private/
Disallow: /admin/

User-agent: OwlerLite
Disallow: /secret/
`

	tests := []struct {
		name       string
		ua         string
		path       string
		disallowed bool
	}{
		{"allowed path", "OwlerLite/0.2", "/public/page.html", false},
		{"disallowed by wildcard", "OwlerLite/0.2", "/private/data", true},
		{"disallowed by specific UA", "OwlerLite/0.2", "/secret/stuff", true},
		{"admin blocked", "OwlerLite/0.2", "/admin/dashboard", true},
		{"root allowed", "OwlerLite/0.2", "/", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDisallowed(robotsTxt, tt.ua, tt.path)
			if got != tt.disallowed {
				t.Errorf("isDisallowed(%q) = %v, want %v", tt.path, got, tt.disallowed)
			}
		})
	}
}

// TestComputeSimpleDiff tests the diff computation
func TestComputeSimpleDiff(t *testing.T) {
	oldText := "Line 1\nLine 2\nLine 3"
	newText := "Line 1\nLine 2 modified\nLine 4"

	diff := computeSimpleDiff(oldText, newText)

	// Should have removed "Line 2", "Line 3" and added "Line 2 modified", "Line 4"
	if len(diff) < 2 {
		t.Errorf("Expected at least 2 diff entries, got %d", len(diff))
	}

	var hasAdded, hasRemoved bool
	for _, d := range diff {
		if d["type"] == "added" {
			hasAdded = true
		}
		if d["type"] == "removed" {
			hasRemoved = true
		}
	}

	if !hasAdded || !hasRemoved {
		t.Error("Diff should contain both added and removed entries")
	}
}
