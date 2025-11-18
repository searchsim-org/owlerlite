
package main
import (
 "context"; "database/sql"; "encoding/json"; "fmt"; "io"; "log"; "net"; "net/http"; "time"
 "github.com/go-chi/chi/v5"
 _ "github.com/mattn/go-sqlite3"
 "google.golang.org/grpc"; "google.golang.org/grpc/reflection"
 pb "owlite/frontier/proto"
)
type server struct{ pb.UnimplementedFrontierServiceServer; db *sql.DB }
func main(){
 db, err := sql.Open("sqlite3","/data/frontier.db?_fk=1"); must(err)
 _,err = db.Exec(`CREATE TABLE IF NOT EXISTS queue(id TEXT PRIMARY KEY, url TEXT, scope TEXT, priority INTEGER DEFAULT 0, enqueued_at INTEGER DEFAULT (strftime('%s','now')));`); must(err)
 s := &server{db:db}
 lis,err := net.Listen("tcp",":7071"); must(err)
 g := grpc.NewServer(); pb.RegisterFrontierServiceServer(g,s); reflection.Register(g)
 r := chi.NewRouter(); r.Get("/stats", s.stats); r.Post("/seed", s.seed); go func(){ log.Println("REST :7072"); must(http.ListenAndServe(":7072",r)) }()
 log.Println("gRPC :7071"); must(g.Serve(lis))
}
func (s *server) Put(ctx context.Context, req *pb.PutRequest)(*pb.PutResponse,error){
 tx,_ := s.db.Begin(); st,_ := tx.Prepare(`INSERT OR IGNORE INTO queue(id,url,scope,priority) VALUES(?,?,?,?)`)
 acc:=0; for _,it:= range req.Items{ if it.Id==""{ it.Id=fmt.Sprintf("%d", time.Now().UnixNano()) }; if _,err:=st.Exec(it.Id,it.Url,it.Scope,it.Priority); err==nil{ acc++ } }
 tx.Commit(); return &pb.PutResponse{Accepted:int32(acc)}, nil
}
func (s *server) GetNext(req *pb.GetNextRequest, stream pb.FrontierService_GetNextServer) error{
 rows,err := s.db.Query(`SELECT id,url,scope,priority FROM queue WHERE scope=? ORDER BY priority DESC, enqueued_at ASC LIMIT ?`, req.Scope, req.Limit)
 if err!=nil{ return err }; defer rows.Close()
 for rows.Next(){ var id,u,sc string; var pr int64; rows.Scan(&id,&u,&sc,&pr); if err:=stream.Send(&pb.UrlItem{Id:id,Url:u,Scope:sc,Priority:pr}); err!=nil{ return err } }
 return nil
}
func (s *server) Ack(ctx context.Context, req *pb.AckRequest)(*pb.AckResponse,error){
 tx,_ := s.db.Begin(); st,_ := tx.Prepare(`DELETE FROM queue WHERE id=?`); ak:=0
 for _,id := range req.Ids { if _,err := st.Exec(id); err==nil { ak++ } }
 tx.Commit(); return &pb.AckResponse{Acked:int32(ak)}, nil
}
func (s *server) stats(w http.ResponseWriter, r *http.Request){ var c int; _=s.db.QueryRow(`SELECT COUNT(*) FROM queue`).Scan(&c); w.Header().Set("content-type","application/json"); w.Write([]byte(fmt.Sprintf(`{"queued":%d}`,c))) }
func (s *server) seed(w http.ResponseWriter, r *http.Request){
 type In struct{ Scope string `json:"scope"`; Urls []string `json:"urls"` }
 var in In; if err:=json.NewDecoder(io.LimitReader(r.Body,1<<20)).Decode(&in); err!=nil{ http.Error(w,err.Error(),400); return }
 items := []*pb.UrlItem{}; for _,u := range in.Urls { items = append(items, &pb.UrlItem{Url:u, Scope:in.Scope, Id: fmt.Sprintf("%d", time.Now().UnixNano())}) }
 _,err := s.Put(r.Context(), &pb.PutRequest{Items:items}); if err!=nil{ http.Error(w, err.Error(), 500); return }; w.WriteHeader(204)
}
func must(err error){ if err!=nil{ log.Fatal(err) } }
