
package main
import (
 "bytes"; "context"; "crypto/sha1"; "database/sql"; "encoding/hex"; "encoding/json"; "fmt"; "io"; "log"; "math"; "mime/multipart"; "net/http"; "net/url"; "os"; "strings"; "sync"; "time"
 _ "github.com/mattn/go-sqlite3"
 "github.com/PuerkitoBio/goquery"
 "github.com/mfonda/simhash"
 "github.com/go-chi/chi/v5"; "github.com/go-chi/cors"
 "github.com/go-shiori/go-readability"
 "google.golang.org/grpc"
 pb "owlite/frontier/proto"
)

// Thresholds per paper: τ₁=0.97 (unchanged), τ₂=0.90 (changed)
// σ = 1 - Hamming/64, so τ₁=0.97 → Hamming≤2, τ₂=0.90 → Hamming≥6
const (
 TAU1_HAMMING = 2  // Hamming <= 2 means σ >= 0.97 (unchanged)
 TAU2_HAMMING = 6  // Hamming >= 6 means σ <= 0.90 (definitely changed)
)

// robots.txt cache
var (
 robotsCache = make(map[string]bool) // host -> allows crawling
 robotsMu    sync.RWMutex
)

func main(){
 front := env("FRONTIER_GRPC_ADDR","localhost:7071"); lightrag := env("LIGHTRAG_BASE_URL","http://localhost:9621")
 dbp := env("SQLITE_PATH","/data/crawler.db"); ua := env("USER_AGENT","OwlerLite/0.2"); delay := atoi(env("CRAWL_DELAY_MS","500"))
 embURL := env("EMBEDDING_URL", lightrag) // Can use separate lightweight embedder
 embModel := env("EMBEDDING_MODEL", "text-embedding-3-small")
 useLightRAGEmbed := env("USE_LIGHTRAG_EMBED", "false") == "true"
 db, _ := sql.Open("sqlite3", dbp)
 db.Exec(`CREATE TABLE IF NOT EXISTS pages(url TEXT PRIMARY KEY, etag TEXT, last_modified TEXT, last_seen INTEGER);
          CREATE TABLE IF NOT EXISTS chunks(docid TEXT PRIMARY KEY, url TEXT, scope TEXT, chunk_id INTEGER, simhash INTEGER, updated_at INTEGER, embedding BLOB);
          CREATE TABLE IF NOT EXISTS versions(url TEXT, chunk_id INTEGER, version_ts INTEGER, text TEXT, PRIMARY KEY(url,chunk_id,version_ts));
          CREATE TABLE IF NOT EXISTS robots_blocked(host TEXT PRIMARY KEY, blocked_at INTEGER);`)
 go serveHTTP(db)
 conn, _ := grpc.Dial(front, grpc.WithInsecure()); defer conn.Close()
 client := pb.NewFrontierServiceClient(conn)
 cfg := &crawlConfig{lightrag: lightrag, ua: ua, embURL: embURL, embModel: embModel, useLightRAGEmbed: useLightRAGEmbed}
 for { // poll
   stream, err := client.GetNext(context.Background(), &pb.GetNextRequest{Scope: "default", Limit: 8})
   if err != nil { time.Sleep(2*time.Second); continue }
   items := []*pb.UrlItem{}; for { it,err := stream.Recv(); if err==io.EOF { break }; if err!=nil{ break }; items = append(items, it) }
   if len(items)==0 { time.Sleep(time.Duration(delay)*time.Millisecond); continue }
   var ack []string
   for _, it := range items { if err := crawlOne(db, cfg, it); err==nil { ack = append(ack, it.Id) } else { log.Printf("crawl %s: %v", it.Url, err) }; time.Sleep(time.Duration(delay)*time.Millisecond) }
   client.Ack(context.Background(), &pb.AckRequest{Ids:ack})
 }
}

type crawlConfig struct {
 lightrag string
 ua string
 embURL string
 embModel string
 useLightRAGEmbed bool
}

func serveHTTP(db *sql.DB){
 r := chi.NewRouter(); r.Use(cors.Handler(cors.Options{AllowedOrigins:[]string{"*"},AllowedMethods:[]string{"GET"},AllowedHeaders:[]string{"*"}}))
 r.Get("/versions", func(w http.ResponseWriter, r *http.Request){
   u := r.URL.Query().Get("url"); cid := atoi(r.URL.Query().Get("chunk_id"))
   var rows *sql.Rows; var err error
   if cid>0 { rows,err = db.Query(`SELECT version_ts,text,chunk_id FROM versions WHERE url=? AND chunk_id=? ORDER BY version_ts DESC LIMIT 10`, u, cid)
   } else { rows,err = db.Query(`SELECT version_ts,text,chunk_id FROM versions WHERE url=? ORDER BY version_ts DESC LIMIT 50`, u) }
   if err != nil { http.Error(w, err.Error(), 500); return }
   defer rows.Close()
   type V struct{ VersionTS int64 `json:"version_ts"`; Text string `json:"text"`; ChunkID int `json:"chunk_id"` }
   var out []V; for rows.Next(){ var v V; rows.Scan(&v.VersionTS,&v.Text,&v.ChunkID); out = append(out, v) }
   w.Header().Set("content-type","application/json"); json.NewEncoder(w).Encode(map[string]any{"versions": out})
 })
 r.Get("/diff", func(w http.ResponseWriter, r *http.Request){
   u := r.URL.Query().Get("url"); cid := atoi(r.URL.Query().Get("chunk_id"))
   rows,err := db.Query(`SELECT version_ts,text FROM versions WHERE url=? AND chunk_id=? ORDER BY version_ts DESC LIMIT 2`, u, cid)
   if err != nil { http.Error(w, err.Error(), 500); return }
   defer rows.Close()
   var versions []struct{ TS int64; Text string }
   for rows.Next(){ var ts int64; var txt string; rows.Scan(&ts, &txt); versions = append(versions, struct{TS int64; Text string}{ts, txt}) }
   if len(versions) < 2 { json.NewEncoder(w).Encode(map[string]any{"diff": nil, "message": "Not enough versions"}); return }
   diff := computeSimpleDiff(versions[1].Text, versions[0].Text)
   w.Header().Set("content-type","application/json"); json.NewEncoder(w).Encode(map[string]any{"old_ts": versions[1].TS, "new_ts": versions[0].TS, "diff": diff})
 })
 r.Get("/stats", func(w http.ResponseWriter, r *http.Request){
   var pages, chunks, versions int
   db.QueryRow(`SELECT COUNT(*) FROM pages`).Scan(&pages)
   db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&chunks)
   db.QueryRow(`SELECT COUNT(*) FROM versions`).Scan(&versions)
   w.Header().Set("content-type","application/json")
   json.NewEncoder(w).Encode(map[string]any{"pages": pages, "chunks": chunks, "versions": versions})
 })
 log.Println("Crawler HTTP :8081"); http.ListenAndServe(":8081", r)
}

// Simple diff: returns added/removed lines
func computeSimpleDiff(oldText, newText string) []map[string]string {
 oldLines := strings.Split(oldText, "\n")
 newLines := strings.Split(newText, "\n")
 oldSet := make(map[string]bool)
 for _, l := range oldLines { oldSet[l] = true }
 newSet := make(map[string]bool)
 for _, l := range newLines { newSet[l] = true }
 var diff []map[string]string
 for _, l := range oldLines { if !newSet[l] && strings.TrimSpace(l) != "" { diff = append(diff, map[string]string{"type": "removed", "line": l}) } }
 for _, l := range newLines { if !oldSet[l] && strings.TrimSpace(l) != "" { diff = append(diff, map[string]string{"type": "added", "line": l}) } }
 return diff
}

// robots.txt compliance
func canCrawl(ua, rawURL string) bool {
 parsed, err := url.Parse(rawURL)
 if err != nil { return true }
 host := parsed.Host
 robotsMu.RLock()
 if allowed, ok := robotsCache[host]; ok { robotsMu.RUnlock(); return allowed }
 robotsMu.RUnlock()
 // Fetch robots.txt
 robotsURL := parsed.Scheme + "://" + host + "/robots.txt"
 cli := &http.Client{Timeout: 5*time.Second}
 resp, err := cli.Get(robotsURL)
 allowed := true
 if err == nil && resp.StatusCode == 200 {
   defer resp.Body.Close()
   body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
   allowed = !isDisallowed(string(body), ua, parsed.Path)
 }
 robotsMu.Lock()
 robotsCache[host] = allowed
 robotsMu.Unlock()
 if !allowed { log.Printf("robots.txt blocks %s", rawURL) }
 return allowed
}

// Simple robots.txt parser
func isDisallowed(robotsTxt, ua, path string) bool {
 lines := strings.Split(robotsTxt, "\n")
 inUserAgent := false
 for _, line := range lines {
   line = strings.TrimSpace(line)
   if strings.HasPrefix(strings.ToLower(line), "user-agent:") {
     agent := strings.TrimSpace(line[11:])
     inUserAgent = agent == "*" || strings.Contains(strings.ToLower(ua), strings.ToLower(agent))
   } else if inUserAgent && strings.HasPrefix(strings.ToLower(line), "disallow:") {
     disallowed := strings.TrimSpace(line[9:])
     if disallowed != "" && strings.HasPrefix(path, disallowed) { return true }
   }
 }
 return false
}

func crawlOne(db *sql.DB, cfg *crawlConfig, it *pb.UrlItem) error {
 u := it.Url
 // Check robots.txt
 if !canCrawl(cfg.ua, u) { return fmt.Errorf("blocked by robots.txt") }
 req,_ := http.NewRequest("GET", u, nil); req.Header.Set("User-Agent", cfg.ua)
 var et, lm string; _ = db.QueryRow(`SELECT etag,last_modified FROM pages WHERE url=?`, u).Scan(&et,&lm)
 if et!=""{ req.Header.Set("If-None-Match", et) }; if lm!=""{ req.Header.Set("If-Modified-Since", lm) }
 cli := &http.Client{Timeout:30*time.Second}; resp, err := cli.Do(req); if err!=nil { return err }; defer resp.Body.Close()
 if resp.StatusCode==304 { db.Exec(`UPDATE pages SET last_seen=? WHERE url=?`, time.Now().Unix(), u); return nil }
 if resp.StatusCode<200 || resp.StatusCode>=300 { return fmt.Errorf("status %d", resp.StatusCode) }
 if e:=resp.Header.Get("ETag"); e!=""{ et=e }; if l:=resp.Header.Get("Last-Modified"); l!=""{ lm=l }
 writeWARC(u, resp)
 body, _ := io.ReadAll(resp.Body)
 parsedURL, _ := url.Parse(u)
 art, err := readability.FromReader(bytes.NewReader(body), parsedURL)
 var txt string
 if err==nil && strings.TrimSpace(art.TextContent)!=""{ txt=art.TextContent } else {
   doc, e := goquery.NewDocumentFromReader(bytes.NewReader(body)); if e!=nil { txt = string(body) } else { txt = strings.Join(strings.Fields(doc.Text())," ") }
 }
 chunks := splitByHeadings(txt, 1600)
 for idx, ch := range chunks {
   h := simhash.Simhash(simhash.NewWordFeatureSet([]byte(ch)))
   var prev uint64; var prevText string
   _ = db.QueryRow(`SELECT simhash FROM chunks WHERE url=? AND chunk_id=?`, u, idx).Scan(&prev)
   _ = db.QueryRow(`SELECT text FROM versions WHERE url=? AND chunk_id=? ORDER BY version_ts DESC LIMIT 1`, u, idx).Scan(&prevText)
   
   shouldIndex := false
   if prev == 0 {
     // New chunk
     shouldIndex = true
   } else {
     dist := hammingDistance(h, prev)
     if dist <= TAU1_HAMMING {
       // σ >= 0.97: unchanged, skip
       continue
     } else if dist >= TAU2_HAMMING {
       // σ <= 0.90: definitely changed
       shouldIndex = true
     } else {
       // Intermediate band: use embedding-based SemDeDup
       if prevText != "" && !semanticallySimilar(cfg, prevText, ch) {
         shouldIndex = true
       }
       // If semantically similar, skip re-indexing
     }
   }
   
   if shouldIndex {
     docid := makeDocID(it.Scope, u, idx, time.Now())
     if err := postToLightRAG(cfg.lightrag, it.Scope, u, idx, ch, docid); err==nil {
       db.Exec(`INSERT OR REPLACE INTO chunks(docid,url,scope,chunk_id,simhash,updated_at) VALUES(?,?,?,?,?,?)`, docid,u,it.Scope,idx,h,time.Now().Unix())
       db.Exec(`INSERT OR REPLACE INTO versions(url,chunk_id,version_ts,text) VALUES(?,?,?,?)`, u, idx, time.Now().Unix(), ch)
       log.Printf("indexed chunk %d of %s (hamming=%d)", idx, u, hammingDistance(h, prev))
     }
   }
 }
 db.Exec(`INSERT OR REPLACE INTO pages(url,etag,last_modified,last_seen) VALUES(?,?,?,?)`, u, et, lm, time.Now().Unix())
 return nil
}

// SemDeDup: embedding-based semantic similarity check
func semanticallySimilar(cfg *crawlConfig, oldText, newText string) bool {
 oldEmb := getEmbedding(cfg, oldText)
 newEmb := getEmbedding(cfg, newText)
 if len(oldEmb) == 0 || len(newEmb) == 0 { return false }
 sim := cosineSimilarity(oldEmb, newEmb)
 // If similarity > 0.95, consider semantically equivalent (near-duplicate)
 return sim > 0.95
}

func getEmbedding(cfg *crawlConfig, text string) []float64 {
 // Truncate long texts for efficiency
 if len(text) > 2000 { text = text[:2000] }
 var endpoint string
 if cfg.useLightRAGEmbed {
   endpoint = cfg.lightrag + "/embeddings"
 } else {
   endpoint = cfg.embURL + "/v1/embeddings"
 }
 payload := map[string]any{"input": text, "model": cfg.embModel}
 body, _ := json.Marshal(payload)
 req, _ := http.NewRequest("POST", endpoint, bytes.NewReader(body))
 req.Header.Set("Content-Type", "application/json")
 if apiKey := env("EMBEDDING_API_KEY", ""); apiKey != "" {
   req.Header.Set("Authorization", "Bearer "+apiKey)
 }
 cli := &http.Client{Timeout: 30*time.Second}
 resp, err := cli.Do(req)
 if err != nil { log.Printf("embedding error: %v", err); return nil }
 defer resp.Body.Close()
 if resp.StatusCode != 200 { return nil }
 var result struct {
   Data []struct { Embedding []float64 `json:"embedding"` } `json:"data"`
 }
 json.NewDecoder(resp.Body).Decode(&result)
 if len(result.Data) > 0 { return result.Data[0].Embedding }
 return nil
}

func cosineSimilarity(a, b []float64) float64 {
 if len(a) != len(b) || len(a) == 0 { return 0 }
 var dot, normA, normB float64
 for i := range a {
   dot += a[i] * b[i]
   normA += a[i] * a[i]
   normB += b[i] * b[i]
 }
 if normA == 0 || normB == 0 { return 0 }
 return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func writeWARC(u string, resp *http.Response) error {
 os.MkdirAll("/warc",0o755)
 f,err := os.Create(fmt.Sprintf("/warc/%d.txt", time.Now().UnixNano()))
 if err!=nil { return nil }
 defer f.Close()
 fmt.Fprintf(f, "URL: %s\nDate: %s\nStatus: %d\n\n", u, time.Now().Format(time.RFC3339), resp.StatusCode)
 return nil
}

func splitByHeadings(text string, maxLen int) []string{
 lines := strings.Split(text,"\n"); var out []string; var cur []string; l:=0
 for _, ln := range lines {
   t := strings.TrimSpace(ln); if t=="" { continue }
   if strings.HasPrefix(t,"#") || l+len(t)>maxLen { if len(cur)>0 { out = append(out, strings.Join(cur,"\n")); cur=nil; l=0 } }
   cur = append(cur,t); l += len(t)
 }
 if len(cur)>0 { out = append(out, strings.Join(cur,"\n")) }
 if len(out)==0 { return []string{text} } ; return out
}

func postToLightRAG(base, scope, raw string, idx int, content, docid string) error {
 meta := fmt.Sprintf("---\nscope: %s\nurl: %s\nchunk_id: %d\ndocid: %s\nversion_ts: %d\n---\n\n", scope, raw, idx, docid, time.Now().Unix())
 body := &bytes.Buffer{}; mw := multipart.NewWriter(body); fw,_ := mw.CreateFormFile("file", "doc.md"); fw.Write([]byte(meta+content)); mw.Close()
 req,_ := http.NewRequest("POST", base+"/documents/file", body); req.Header.Set("Content-Type", mw.FormDataContentType())
 cli := &http.Client{Timeout:120*time.Second}; resp, err := cli.Do(req); if err!=nil { return err }; defer resp.Body.Close()
 if resp.StatusCode<200 || resp.StatusCode>=300 { b,_:=io.ReadAll(resp.Body); return fmt.Errorf("lightrag %d: %s", resp.StatusCode, string(b)) }
 return nil
}

func makeDocID(scope, raw string, idx int, t time.Time) string { return fmt.Sprintf("%s_%s_%d_%d", scope, hashURL(raw), idx, t.Unix()) }
func hashURL(u string) string { h := sha1.Sum([]byte(u)); return hex.EncodeToString(h[:8]) }
func env(k,d string) string { if v:=os.Getenv(k); v!=""{ return v }; return d }
func atoi(s string) int { n:=0; for _,r:= range s { if r>='0'&&r<='9' { n=n*10+int(r-'0') } }; return n }
func hammingDistance(h1, h2 uint64) int { x := h1 ^ h2; c := 0; for x != 0 { c++; x &= x - 1 }; return c }

