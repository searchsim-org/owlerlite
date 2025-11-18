
package main
import (
 "database/sql"; "encoding/json"; "fmt"; "io"; "log"; "math"; "net/http"; "os"; "regexp"; "sort"; "strings"; "time"
 "github.com/go-chi/chi/v5"; "github.com/go-chi/cors"
 _ "github.com/mattn/go-sqlite3"
)
type Scope struct{ ID int64 `json:"id"`; Name string `json:"name"`; Patterns []string `json:"patterns"` }
type Item struct{ DocID, Source, Snippet string; Base, Final float64; VersionTS int64; Scope, URL string; ChunkID int }
func main(){
 db,_ := sql.Open("sqlite3", env("SQLITE_PATH","/data/owlerlite.db"))
 db.Exec(`CREATE TABLE IF NOT EXISTS scopes(id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT UNIQUE);
          CREATE TABLE IF NOT EXISTS scope_patterns(scope_id INTEGER, pattern TEXT);
          CREATE TABLE IF NOT EXISTS scope_entities(scope_id INTEGER, entity TEXT);
          CREATE TABLE IF NOT EXISTS impressions(scope_id INTEGER, docid TEXT, n INTEGER DEFAULT 0, PRIMARY KEY(scope_id, docid));
          CREATE TABLE IF NOT EXISTS clicks(scope_id INTEGER, docid TEXT, n INTEGER DEFAULT 0, PRIMARY KEY(scope_id, docid));`)
 lr := env("LIGHTRAG_BASE_URL","http://localhost:9621"); cr := env("CRAWLER_BASE","http://localhost:8081")
 A := atof(env("RAG_SCOPE_ALPHA","0.7")); D := atof(env("RAG_FRESH_DELTA","0.2")); G := atof(env("RAG_KG_GAMMA","0.15"))
 r := chi.NewRouter(); r.Use(cors.Handler(cors.Options{AllowedOrigins:[]string{"*"},AllowedMethods:[]string{"GET","POST"},AllowedHeaders:[]string{"*"}}))
 r.Post("/scopes", func(w http.ResponseWriter, r *http.Request){
   var in struct{ Name string; Patterns []string }; decode(r.Body, &in)
   res,_ := db.Exec(`INSERT INTO scopes(name) VALUES(?)`, in.Name); id,_ := res.LastInsertId()
   for _,p := range in.Patterns { db.Exec(`INSERT INTO scope_patterns(scope_id,pattern) VALUES(?,?)`, id, p) }
   json.NewEncoder(w).Encode(Scope{ID:id, Name:in.Name, Patterns:in.Patterns})
 })
 r.Post("/scopes/{id}/entities", func(w http.ResponseWriter, r *http.Request){
   id := atoi64(chi.URLParam(r,"id")); var in struct{ Entities []string }
   decode(r.Body,&in); for _,e:= range in.Entities { db.Exec(`INSERT INTO scope_entities(scope_id,entity) VALUES(?,?)`, id, strings.TrimSpace(e)) }
   w.WriteHeader(204)
 })
 r.Post("/scopes/{id}/seed", func(w http.ResponseWriter, r *http.Request){
   id := chi.URLParam(r,"id"); var in struct{ Urls []string }; decode(r.Body,&in)
   body,_ := json.Marshal(map[string]any{"scope":id,"urls":in.Urls})
   http.Post("http://frontier:7072/seed","application/json", strings.NewReader(string(body))); w.WriteHeader(204)
 })
 r.Post("/feedback/click", func(w http.ResponseWriter, r *http.Request){
   var in struct{ ScopeID int64 `json:"scope_id"`; DocID string `json:"doc_id"` }; decode(r.Body,&in)
   db.Exec(`INSERT INTO clicks(scope_id,docid,n) VALUES(?,?,1) ON CONFLICT(scope_id,docid) DO UPDATE SET n=n+1`, in.ScopeID, in.DocID); w.WriteHeader(204)
 })
 r.Post("/query", func(w http.ResponseWriter, r *http.Request){
   var in struct{ Query string; ScopeIDs []int64; Mode string }; decode(r.Body,&in); if in.Mode==""{ in.Mode="hybrid" }
   payload,_ := json.Marshal(map[string]any{"query":in.Query,"mode":in.Mode})
   resp,err := http.Post(lr+"/query/data","application/json", strings.NewReader(string(payload))); if err!=nil{ http.Error(w,err.Error(),502); return }
   defer resp.Body.Close(); var data map[string]any; json.NewDecoder(resp.Body).Decode(&data)
   items := extractItems(data)
   now := time.Now().Unix(); ents := loadScopeEntities(db, in.ScopeIDs)
   for i := range items {
     for _, sid := range in.ScopeIDs { db.Exec(`INSERT INTO impressions(scope_id,docid,n) VALUES(?,?,1) ON CONFLICT(scope_id,docid) DO UPDATE SET n=n+1`, sid, items[i].DocID) }
     sid := inferScopeID(items[i].DocID); ctr := scopeCTR(db, sid, items[i].DocID)
     inSel := contains64(in.ScopeIDs, sid); sb := 0.0; if inSel { sb += 0.5 }; sb += ctr
     fresh := freshness(items[i].VersionTS, now)
     m := 0; for _, e := range extractEntities(items[i].Snippet) { if ents[strings.ToLower(e)] { m++ } }
     kg := 0.0; if m>0 { kg = 1.0 - 1.0/math.Log(float64(m)+2) }
     items[i].Final = items[i].Base * (1.0 + A*sb + G*kg) + D*fresh
   }
   sort.SliceStable(items, func(i,j int)bool{ return items[i].Final > items[j].Final })
   json.NewEncoder(w).Encode(map[string]any{"items":items})
 })
 r.Get("/versions", func(w http.ResponseWriter, r *http.Request){
   u := r.URL.Query().Get("url"); cid := r.URL.Query().Get("chunk_id")
   resp,err := http.Get(fmt.Sprintf("%s/versions?url=%s&chunk_id=%s", cr, urlq(u), urlq(cid))); if err!=nil{ http.Error(w,err.Error(),502); return }
   defer resp.Body.Close(); io.Copy(w, resp.Body)
 })
 log.Println("Orchestrator :7000"); http.ListenAndServe(":7000", r)
}
func extractItems(data map[string]any) []Item {
 out := []Item{}; add := func(t string, rank int){
   d,s,u,c,v := parseFrontMatter(t); if u==""{ u="unknown" }
   out = append(out, Item{DocID:d,Scope:s,URL:u,ChunkID:c,VersionTS:v,Snippet:stripFrontMatter(t),Base:1.0/float64(rank+1),Source:u})
 }
 if cs,ok := data["chunks"].([]any); ok { for i, c := range cs { if m,ok:=c.(map[string]any); ok { if t,ok:=m["text"].(string); ok { add(t,i) } } }; return out }
 if ps,ok := data["passages"].([]any); ok { for i, p := range ps { if m,ok:=p.(map[string]any); ok { if t,ok:=m["text"].(string); ok { add(t,i) } } } }
 return out
}
func stripFrontMatter(s string) string { if strings.HasPrefix(s,"---"){ if idx:=strings.Index(s[3:],"---"); idx>=0 { return s[3+idx+3:] } }; return s }
func parseFrontMatter(s string) (doc, sc, url string, cid int, ver int64){
 if !strings.HasPrefix(s,"---"){ return "","","",0,0 }; end := strings.Index(s[3:],"---"); if end<0 { return "","","",0,0 }
 meta := s[3:3+end]; for _, ln := range strings.Split(meta,"\n"){ kv := strings.SplitN(strings.TrimSpace(ln),":",2); if len(kv)!=2 { continue }
   k:=strings.TrimSpace(kv[0]); v:=strings.TrimSpace(kv[1])
   switch k { case "docid": doc=v; case "scope": sc=v; case "url": url=v; case "chunk_id": fmt.Sscanf(v,"%d",&cid); case "version_ts": fmt.Sscanf(v,"%d",&ver) }
 }
 return
}
func inferScopeID(docid string) int64 { var sid int64; fmt.Sscanf(docid, "%d_", &sid); return sid }
var entRe = regexp.MustCompile(`\b([A-Z][a-z]+(?:\s[A-Z][a-z]+){0,2})\b`)
func extractEntities(s string) []string { m:=entRe.FindAllString(s,-1); seen:=map[string]struct{}{}; out:=[]string{}; for _,x:= range m { x=strings.TrimSpace(x); xl:=strings.ToLower(x); if len(x)<3{continue}; if _,ok:=seen[xl]; ok{continue}; seen[xl]=struct{}{}; out=append(out,x) }; return out }
func loadScopeEntities(db *sql.DB, scopes []int64) map[string]bool {
 out := map[string]bool{}; if len(scopes)==0 { return out }
 q := "SELECT entity FROM scope_entities WHERE scope_id IN ("+placeholders(len(scopes))+")"
 args := []any{}; for _, s := range scopes { args = append(args, s) }
 rows,err := db.Query(q, args...); if err!=nil { return out }; defer rows.Close()
 for rows.Next(){ var e string; rows.Scan(&e); out[strings.ToLower(strings.TrimSpace(e))]=true }
 return out
}
func scopeCTR(db *sql.DB, sid int64, docid string) float64 {
 if sid==0 { return 0 }; var imp,clk int64
 _ = db.QueryRow(`SELECT COALESCE(n,0) FROM impressions WHERE scope_id=? AND docid=?`, sid, docid).Scan(&imp)
 _ = db.QueryRow(`SELECT COALESCE(n,0) FROM clicks WHERE scope_id=? AND docid=?`, sid, docid).Scan(&clk)
 return float64(clk+1)/float64(imp+2)
}
func freshness(ts, now int64) float64 { if ts==0 { return 0 }; age:= now-ts; return 1.0/(1.0+math.Log(2+float64(age)/86400.0)) }
func env(k,d string) string { if v:=os.Getenv(k); v!=""{ return v }; return d }
func atof(s string) float64 { var f float64; fmt.Sscanf(s,"%f",&f); return f }
func decode(r io.Reader, v any) error { return json.NewDecoder(io.LimitReader(r,1<<20)).Decode(v) }
func urlq(s string) string { return strings.ReplaceAll(s," ","%20") }
func contains64(xs []int64, v int64) bool { for _,x := range xs { if x==v { return true } }; return false }
func atoi64(s string) int64 { var n int64; fmt.Sscanf(s,"%d",&n); return n }
func placeholders(n int) string { if n<=0 { return "" }; return strings.TrimRight(strings.Repeat("?,",n),",") }
