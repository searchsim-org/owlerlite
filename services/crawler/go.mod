module owlerlite/crawler

go 1.22

require (
	github.com/PuerkitoBio/goquery v1.9.2
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.1
	github.com/go-shiori/go-readability v0.0.0-20230421032831-c66949dfc0ad
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/mfonda/simhash v0.0.0-20151007195837-79f94a1100d6
	google.golang.org/grpc v1.66.0
	owlite/frontier/proto v0.0.0-00010101000000-000000000000
)

require (
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/go-shiori/dom v0.0.0-20210627111528-4e4722cd0d65 // indirect
	github.com/gogs/chardet v0.0.0-20211120154057-b7413eaefb8f // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240604185151-ef581f913117 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace owlite/frontier/proto => ../frontier/proto
