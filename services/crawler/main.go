
package main
import (
 "bytes"; "context"; "crypto/sha1"; "database/sql"; "encoding/hex"; "encoding/json"; "fmt"; "io"; "log"; "mime/multipart"; "net/http"; "net/url"; "os"; "strings"; "time"
 _ "github.com/mattn/go-sqlite3"
 "github.com/PuerkitoBio/goquery"
 "github.com/mfonda/simhash"
 "github.com/go-chi/chi/v5"; "github.com/go-chi/cors"
 "github.com/go-shiori/go-readability"
 "google.golang.org/grpc"
 pb "owlite/frontier/proto"
)
func main(){
 front := env("FRONTIER_GRPC_ADDR","localhost:7071"); lightrag := env("LIGHTRAG_BASE_URL","http://localhost:9621")
 dbp := env("SQLITE_PATH","/data/crawler.db"); ua := env("USER_AGENT","OwlerLite/0.2"); delay := atoi(env("CRAWL_DELAY_MS","500"))
 db, _ := sql.Open("sqlite3", dbp)
 db.Exec(`CREATE TABLE IF NOT EXISTS pages(url TEXT PRIMARY KEY, etag TEXT, last_modified TEXT, last_seen INTEGER);
          CREATE TABLE IF NOT EXISTS chunks(docid TEXT PRIMARY KEY, url TEXT, scope TEXT, chunk_id INTEGER, simhash INTEGER, updated_at INTEGER);
          CREATE TABLE IF NOT EXISTS versions(url TEXT, chunk_id INTEGER, version_ts INTEGER, text TEXT, PRIMARY KEY(url,chunk_id,version_ts));`)
 go serveHTTP(db)
 conn, _ := grpc.Dial(front, grpc.WithInsecure()); defer conn.Close()
 client := pb.NewFrontierServiceClient(conn)
 for { // poll
   stream, err := client.GetNext(context.Background(), &pb.GetNextRequest{Scope: "default", Limit: 8})
   if err != nil { time.Sleep(2*time.Second); continue }
   items := []*pb.UrlItem{}; for { it,err := stream.Recv(); if err==io.EOF { break }; if err!=nil{ break }; items = append(items, it) }
   if len(items)==0 { time.Sleep(time.Duration(delay)*time.Millisecond); continue }
   var ack []string
   for _, it := range items { if err := crawlOne(db, ua, lightrag, it); err==nil { ack = append(ack, it.Id) }; time.Sleep(time.Duration(delay)*time.Millisecond) }
   client.Ack(context.Background(), &pb.AckRequest{Ids:ack})
 }
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
 log.Println("Crawler versions HTTP :8081"); http.ListenAndServe(":8081", r)
}
func crawlOne(db *sql.DB, ua, lightrag string, it *pb.UrlItem) error {
 u := it.Url
 req,_ := http.NewRequest("GET", u, nil); req.Header.Set("User-Agent", ua)
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
   var prev uint64; _ = db.QueryRow(`SELECT simhash FROM chunks WHERE url=? AND chunk_id=?`, u, idx).Scan(&prev)
   if prev==0 || hammingDistance(h, prev) >= 3 {
     docid := makeDocID(it.Scope, u, idx, time.Now())
     if err := postToLightRAG(lightrag, it.Scope, u, idx, ch, docid); err==nil {
       db.Exec(`INSERT OR REPLACE INTO chunks(docid,url,scope,chunk_id,simhash,updated_at) VALUES(?,?,?,?,?,?)`, docid,u,it.Scope,idx,h,time.Now().Unix())
       db.Exec(`INSERT OR REPLACE INTO versions(url,chunk_id,version_ts,text) VALUES(?,?,?,?)`, u, idx, time.Now().Unix(), ch)
     }
   }
 }
 db.Exec(`INSERT OR REPLACE INTO pages(url,etag,last_modified,last_seen) VALUES(?,?,?,?)`, u, et, lm, time.Now().Unix())
 return nil
}
func writeWARC(u string, resp *http.Response) error {
 // WARC writing simplified - stores basic metadata
 os.MkdirAll("/warc",0o755)
 f,err := os.Create(fmt.Sprintf("/warc/%d.txt", time.Now().UnixNano()))
 if err!=nil { return nil } // Non-critical, continue
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
