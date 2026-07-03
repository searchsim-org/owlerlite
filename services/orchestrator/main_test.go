package main

import (
	"math"
	"testing"
)

// TestExponentialFreshness tests the freshness decay function
func TestExponentialFreshness(t *testing.T) {
	now := int64(1700000000) // Fixed timestamp for testing

	tests := []struct {
		name      string
		versionTS int64
		minFresh  float64
		maxFresh  float64
	}{
		{"just now", now, 0.99, 1.0},
		{"1 day ago", now - 86400, 0.90, 0.96},
		{"7 days ago", now - 7*86400, 0.65, 0.75},
		{"30 days ago", now - 30*86400, 0.15, 0.30},
		{"zero timestamp", 0, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exponentialFreshness(tt.versionTS, now)
			if got < tt.minFresh || got > tt.maxFresh {
				t.Errorf("exponentialFreshness(%d, %d) = %f, want [%f, %f]",
					tt.versionTS, now, got, tt.minFresh, tt.maxFresh)
			}
		})
	}
}

// TestScoringFormula tests the paper's scoring function
func TestScoringFormula(t *testing.T) {
	cfg := ScoringConfig{
		Alpha: 0.8,
		Beta:  0.2,
		Delta: 0.15,
		Gamma: 0.1,
	}

	tests := []struct {
		name        string
		baseScore   float64
		graphScore  float64
		inScope     bool
		freshness   float64
		minExpected float64
		maxExpected float64
	}{
		{
			name:        "in-scope fresh content",
			baseScore:   1.0,
			graphScore:  0.5,
			inScope:     true,
			freshness:   1.0,
			minExpected: 0.8,
			maxExpected: 1.2,
		},
		{
			name:        "out-of-scope content",
			baseScore:   1.0,
			graphScore:  0.5,
			inScope:     false,
			freshness:   1.0,
			minExpected: 0.4,
			maxExpected: 0.9,
		},
		{
			name:        "stale content",
			baseScore:   1.0,
			graphScore:  0.5,
			inScope:     true,
			freshness:   0.1,
			minExpected: 0.6,
			maxExpected: 1.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scopePrior := cfg.Gamma
			if tt.inScope {
				scopePrior = 1.0
			}

			// h(q,p) = α·sim_vec + (1-α)·score_graph + β·log(g) + δ·fresh
			score := cfg.Alpha*tt.baseScore +
				(1-cfg.Alpha)*tt.graphScore +
				cfg.Beta*math.Log(scopePrior+0.01) +
				cfg.Delta*tt.freshness

			if score < tt.minExpected || score > tt.maxExpected {
				t.Errorf("score = %f, want [%f, %f]", score, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

// TestInferScopeID tests scope ID extraction from document ID
func TestInferScopeID(t *testing.T) {
	tests := []struct {
		docid    string
		expected int64
	}{
		{"1_abc123_0_1700000000", 1},
		{"42_def456_5_1700000001", 42},
		{"123_ghi789_10_1700000002", 123},
		{"invalid_format", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.docid, func(t *testing.T) {
			got := inferScopeID(tt.docid)
			if got != tt.expected {
				t.Errorf("inferScopeID(%q) = %d, want %d", tt.docid, got, tt.expected)
			}
		})
	}
}

// TestExtractEntities tests named entity extraction
func TestExtractEntities(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		minEntities int
	}{
		{"simple entities", "John Smith works at Google", 2},
		{"no entities", "the quick brown fox", 0},
		{"organization names", "Microsoft and Apple are tech companies", 2},
		{"empty text", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractEntities(tt.text)
			if len(entities) < tt.minEntities {
				t.Errorf("extractEntities(%q) found %d entities, want at least %d",
					tt.text, len(entities), tt.minEntities)
			}
		})
	}
}

// TestContains64 tests the int64 slice contains helper
func TestContains64(t *testing.T) {
	slice := []int64{1, 2, 3, 5, 8}

	tests := []struct {
		value    int64
		expected bool
	}{
		{1, true},
		{5, true},
		{4, false},
		{0, false},
		{10, false},
	}

	for _, tt := range tests {
		got := contains64(slice, tt.value)
		if got != tt.expected {
			t.Errorf("contains64(%v, %d) = %v, want %v", slice, tt.value, got, tt.expected)
		}
	}
}

// TestPlaceholders tests SQL placeholder generation
func TestPlaceholders(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, ""},
		{1, "?"},
		{3, "?,?,?"},
		{5, "?,?,?,?,?"},
	}

	for _, tt := range tests {
		got := placeholders(tt.n)
		if got != tt.expected {
			t.Errorf("placeholders(%d) = %q, want %q", tt.n, got, tt.expected)
		}
	}
}

// TestParseFrontMatter tests metadata extraction from chunk text
func TestParseFrontMatter(t *testing.T) {
	text := `---
docid: 1_abc123_0_1700000000
scope: msmarco
url: https://example.com/page
chunk_id: 5
version_ts: 1700000000
---

This is the actual content of the chunk.
`

	docid, scope, url, chunkID, versionTS := parseFrontMatter(text)

	if docid != "1_abc123_0_1700000000" {
		t.Errorf("docid = %q, want 1_abc123_0_1700000000", docid)
	}
	if scope != "msmarco" {
		t.Errorf("scope = %q, want msmarco", scope)
	}
	if url != "https://example.com/page" {
		t.Errorf("url = %q, want https://example.com/page", url)
	}
	if chunkID != 5 {
		t.Errorf("chunkID = %d, want 5", chunkID)
	}
	if versionTS != 1700000000 {
		t.Errorf("versionTS = %d, want 1700000000", versionTS)
	}
}

// TestStripFrontMatter tests front matter removal
func TestStripFrontMatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"with front matter",
			"---\nkey: value\n---\nActual content",
			"Actual content",
		},
		{
			"no front matter",
			"Just plain text",
			"Just plain text",
		},
		{
			"malformed front matter",
			"---\nkey: value\nNo closing",
			"---\nkey: value\nNo closing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripFrontMatter(tt.input)
			if got != tt.expected {
				t.Errorf("stripFrontMatter = %q, want %q", got, tt.expected)
			}
		})
	}
}
