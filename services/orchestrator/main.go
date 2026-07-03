package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	_ "github.com/mattn/go-sqlite3"
)

// Scoring weights from paper Section 3
type ScoringConfig struct {
	Alpha float64 // Weight for vector similarity vs graph score
	Beta  float64 // Weight for scope prior log term
	Delta float64 // Weight for freshness
	Gamma float64 // Scope prior for out-of-scope (1.0 for in-scope)
}

type Scope struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Patterns []string `json:"patterns"`
}

// ScoredItem with full breakdown per paper
type ScoredItem struct {
	DocID     string  `json:"doc_id"`
	URL       string  `json:"url"`
	Snippet   string  `json:"snippet"`
	ChunkID   int     `json:"chunk_id"`
	VersionTS int64   `json:"version_ts"`
	Scope     string  `json:"scope"`
	Score     float64 `json:"score"`
	Breakdown struct {
		SimVec     float64 `json:"sim_vec"`
		ScoreGraph float64 `json:"score_graph"`
		ScopePrior float64 `json:"scope_prior"`
		Freshness  float64 `json:"freshness"`
	} `json:"breakdown"`
}

type QueryResponse struct {
	Items     []ScoredItem `json:"items"`
	Answer    string       `json:"answer,omitempty"`
	Rationale string       `json:"rationale,omitempty"`
}

func main() {
	db, _ := sql.Open("sqlite3", env("SQLITE_PATH", "/data/owlerlite.db"))
	db.Exec(`CREATE TABLE IF NOT EXISTS scopes(id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT UNIQUE);
          CREATE TABLE IF NOT EXISTS scope_patterns(scope_id INTEGER, pattern TEXT);
          CREATE TABLE IF NOT EXISTS scope_entities(scope_id INTEGER, entity TEXT);
          CREATE TABLE IF NOT EXISTS impressions(scope_id INTEGER, docid TEXT, n INTEGER DEFAULT 0, PRIMARY KEY(scope_id, docid));
          CREATE TABLE IF NOT EXISTS clicks(scope_id INTEGER, docid TEXT, n INTEGER DEFAULT 0, PRIMARY KEY(scope_id, docid));`)

	lr := env("LIGHTRAG_BASE_URL", "http://localhost:9621")
	cr := env("CRAWLER_BASE", "http://localhost:8081")
	llmURL := env("LLM_BASE_URL", "https://api.openai.com/v1")
	llmModel := env("LLM_MODEL", "gpt-4o-mini")
	llmAPIKey := env("LLM_API_KEY", "")

	cfg := ScoringConfig{
		Alpha: atof(env("RAG_ALPHA", "0.8")),  // Vector vs graph weight
		Beta:  atof(env("RAG_BETA", "0.2")),   // Scope prior weight
		Delta: atof(env("RAG_DELTA", "0.15")), // Freshness weight
		Gamma: atof(env("RAG_GAMMA", "0.1")),  // Out-of-scope prior
	}

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"}, AllowedHeaders: []string{"*"}}))

	// Scope CRUD
	r.Post("/scopes", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Name     string
			Patterns []string
		}
		decode(r.Body, &in)
		res, _ := db.Exec(`INSERT INTO scopes(name) VALUES(?)`, in.Name)
		id, _ := res.LastInsertId()
		for _, p := range in.Patterns {
			db.Exec(`INSERT INTO scope_patterns(scope_id,pattern) VALUES(?,?)`, id, p)
		}
		json.NewEncoder(w).Encode(Scope{ID: id, Name: in.Name, Patterns: in.Patterns})
	})

	r.Get("/scopes", func(w http.ResponseWriter, r *http.Request) {
		rows, _ := db.Query(`SELECT id, name FROM scopes`)
		defer rows.Close()
		var scopes []Scope
		for rows.Next() {
			var s Scope
			rows.Scan(&s.ID, &s.Name)
			// Load patterns
			prows, _ := db.Query(`SELECT pattern FROM scope_patterns WHERE scope_id=?`, s.ID)
			for prows.Next() {
				var p string
				prows.Scan(&p)
				s.Patterns = append(s.Patterns, p)
			}
			prows.Close()
			scopes = append(scopes, s)
		}
		json.NewEncoder(w).Encode(map[string]any{"scopes": scopes})
	})

	r.Post("/scopes/{id}/entities", func(w http.ResponseWriter, r *http.Request) {
		id := atoi64(chi.URLParam(r, "id"))
		var in struct{ Entities []string }
		decode(r.Body, &in)
		for _, e := range in.Entities {
			db.Exec(`INSERT INTO scope_entities(scope_id,entity) VALUES(?,?)`, id, strings.TrimSpace(e))
		}
		w.WriteHeader(204)
	})

	r.Post("/scopes/{id}/seed", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var in struct{ Urls []string }
		decode(r.Body, &in)
		body, _ := json.Marshal(map[string]any{"scope": id, "urls": in.Urls})
		http.Post("http://frontier:7072/seed", "application/json", strings.NewReader(string(body)))
		w.WriteHeader(204)
	})

	r.Post("/feedback/click", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			ScopeID int64  `json:"scope_id"`
			DocID   string `json:"doc_id"`
		}
		decode(r.Body, &in)
		db.Exec(`INSERT INTO clicks(scope_id,docid,n) VALUES(?,?,1) ON CONFLICT(scope_id,docid) DO UPDATE SET n=n+1`, in.ScopeID, in.DocID)
		w.WriteHeader(204)
	})

	// Main query endpoint
	r.Post("/query", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Query      string  `json:"query"`
			ScopeIDs   []int64 `json:"scope_ids"`
			Mode       string  `json:"mode"`
			GenerateNL bool    `json:"generate_nl"`
		}
		decode(r.Body, &in)
		if in.Mode == "" {
			in.Mode = "hybrid"
		}

		// Call LightRAG for base retrieval
		payload, _ := json.Marshal(map[string]any{"query": in.Query, "mode": in.Mode})
		resp, err := http.Post(lr+"/query/data", "application/json", strings.NewReader(string(payload)))
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		defer resp.Body.Close()
		var data map[string]any
		json.NewDecoder(resp.Body).Decode(&data)

		items := extractAndScoreItems(db, cfg, data, in.ScopeIDs)

		// Sort by final score
		sort.SliceStable(items, func(i, j int) bool { return items[i].Score > items[j].Score })

		// Limit to top 10
		if len(items) > 10 {
			items = items[:10]
		}

		response := QueryResponse{Items: items}

		// Generate NL rationale if requested
		if in.GenerateNL && len(items) > 0 && llmAPIKey != "" {
			response.Rationale = generateRationale(llmURL, llmModel, llmAPIKey, in.Query, items)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Version diff proxy
	r.Get("/versions", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("url")
		cid := r.URL.Query().Get("chunk_id")
		resp, err := http.Get(fmt.Sprintf("%s/versions?url=%s&chunk_id=%s", cr, urlq(u), urlq(cid)))
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, resp.Body)
	})

	r.Get("/diff", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("url")
		cid := r.URL.Query().Get("chunk_id")
		resp, err := http.Get(fmt.Sprintf("%s/diff?url=%s&chunk_id=%s", cr, urlq(u), urlq(cid)))
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, resp.Body)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "service": "orchestrator"})
	})

	log.Println("Orchestrator :7000")
	http.ListenAndServe(":7000", r)
}

func extractAndScoreItems(db *sql.DB, cfg ScoringConfig, data map[string]any, scopeIDs []int64) []ScoredItem {
	now := time.Now().Unix()
	ents := loadScopeEntities(db, scopeIDs)

	var items []ScoredItem

	add := func(t string, rank int) {
		docid, scope, url, chunkID, versionTS := parseFrontMatter(t)
		if url == "" {
			url = "unknown"
		}

		snippet := stripFrontMatter(t)
		baseScore := 1.0 / float64(rank+1) // Use rank as proxy for sim_vec

		// Calculate scope prior g(p; S_q)
		sid := inferScopeID(docid)
		inSelectedScope := contains64(scopeIDs, sid)
		scopePrior := cfg.Gamma // Out-of-scope default
		if inSelectedScope {
			scopePrior = 1.0
		}

		// Calculate freshness with exponential decay
		fresh := exponentialFreshness(versionTS, now)

		// Calculate graph score from entity matches
		m := 0
		for _, e := range extractEntities(snippet) {
			if ents[strings.ToLower(e)] {
				m++
			}
		}
		graphScore := 0.0
		if m > 0 {
			graphScore = 1.0 - 1.0/math.Log(float64(m)+2)
		}

		// Combine per paper formula:
		// h(q,p) = α·sim_vec + (1-α)·score_graph + β·log(g) + δ·fresh
		finalScore := cfg.Alpha*baseScore + (1-cfg.Alpha)*graphScore + cfg.Beta*math.Log(scopePrior+0.01) + cfg.Delta*fresh

		item := ScoredItem{
			DocID:     docid,
			URL:       url,
			Snippet:   snippet,
			ChunkID:   chunkID,
			VersionTS: versionTS,
			Scope:     scope,
			Score:     finalScore,
		}
		item.Breakdown.SimVec = baseScore
		item.Breakdown.ScoreGraph = graphScore
		item.Breakdown.ScopePrior = scopePrior
		item.Breakdown.Freshness = fresh

		// Track impressions
		for _, sid := range scopeIDs {
			db.Exec(`INSERT INTO impressions(scope_id,docid,n) VALUES(?,?,1) ON CONFLICT(scope_id,docid) DO UPDATE SET n=n+1`, sid, docid)
		}

		items = append(items, item)
	}

	// Parse chunks or passages from LightRAG response
	if cs, ok := data["chunks"].([]any); ok {
		for i, c := range cs {
			if m, ok := c.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					add(t, i)
				}
			}
		}
	}
	if ps, ok := data["passages"].([]any); ok {
		for i, p := range ps {
			if m, ok := p.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					add(t, i)
				}
			}
		}
	}

	return items
}

// Exponential freshness decay per paper
func exponentialFreshness(ts, now int64) float64 {
	if ts == 0 {
		return 0
	}
	ageDays := float64(now-ts) / 86400.0
	// Exponential decay: fresh content scores higher
	return math.Exp(-0.05 * ageDays)
}

// Generate NL rationale via LLM
func generateRationale(llmURL, model, apiKey, query string, items []ScoredItem) string {
	if len(items) == 0 {
		return ""
	}

	// Build context for rationale
	var sb strings.Builder
	sb.WriteString("Query: " + query + "\n\nTop results:\n")
	for i, item := range items {
		if i >= 3 {
			break
		}
		sb.WriteString(fmt.Sprintf("%d. [Score: %.3f] %s\n   Breakdown - Semantic: %.2f, Graph: %.2f, Scope: %.2f, Fresh: %.2f\n",
			i+1, item.Score, item.URL,
			item.Breakdown.SimVec, item.Breakdown.ScoreGraph, item.Breakdown.ScopePrior, item.Breakdown.Freshness))
	}

	prompt := fmt.Sprintf(`Given this search query and results with their score breakdowns, provide a brief (2-3 sentence) natural language explanation of why these results were selected and ranked this way. Focus on scope relevance and freshness.

%s

Rationale:`, sb.String())

	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  150,
		"temperature": 0.3,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", llmURL+"/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	cli := &http.Client{Timeout: 30 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Choices) > 0 {
		return strings.TrimSpace(result.Choices[0].Message.Content)
	}
	return ""
}

func stripFrontMatter(s string) string {
	if strings.HasPrefix(s, "---") {
		if idx := strings.Index(s[3:], "---"); idx >= 0 {
			return strings.TrimSpace(s[3+idx+3:])
		}
	}
	return s
}

func parseFrontMatter(s string) (doc, sc, url string, cid int, ver int64) {
	if !strings.HasPrefix(s, "---") {
		return "", "", "", 0, 0
	}
	end := strings.Index(s[3:], "---")
	if end < 0 {
		return "", "", "", 0, 0
	}
	meta := s[3 : 3+end]
	for _, ln := range strings.Split(meta, "\n") {
		kv := strings.SplitN(strings.TrimSpace(ln), ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		switch k {
		case "docid":
			doc = v
		case "scope":
			sc = v
		case "url":
			url = v
		case "chunk_id":
			fmt.Sscanf(v, "%d", &cid)
		case "version_ts":
			fmt.Sscanf(v, "%d", &ver)
		}
	}
	return
}

func inferScopeID(docid string) int64 {
	var sid int64
	fmt.Sscanf(docid, "%d_", &sid)
	return sid
}

var entRe = regexp.MustCompile(`\b([A-Z][a-z]+(?:\s[A-Z][a-z]+){0,2})\b`)

func extractEntities(s string) []string {
	m := entRe.FindAllString(s, -1)
	seen := map[string]struct{}{}
	out := []string{}
	for _, x := range m {
		x = strings.TrimSpace(x)
		xl := strings.ToLower(x)
		if len(x) < 3 {
			continue
		}
		if _, ok := seen[xl]; ok {
			continue
		}
		seen[xl] = struct{}{}
		out = append(out, x)
	}
	return out
}

func loadScopeEntities(db *sql.DB, scopes []int64) map[string]bool {
	out := map[string]bool{}
	if len(scopes) == 0 {
		return out
	}
	q := "SELECT entity FROM scope_entities WHERE scope_id IN (" + placeholders(len(scopes)) + ")"
	args := []any{}
	for _, s := range scopes {
		args = append(args, s)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var e string
		rows.Scan(&e)
		out[strings.ToLower(strings.TrimSpace(e))] = true
	}
	return out
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func atof(s string) float64           { var f float64; fmt.Sscanf(s, "%f", &f); return f }
func decode(r io.Reader, v any) error { return json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(v) }
func urlq(s string) string            { return strings.ReplaceAll(s, " ", "%20") }
func contains64(xs []int64, v int64) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
func atoi64(s string) int64 { var n int64; fmt.Sscanf(s, "%d", &n); return n }
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}
