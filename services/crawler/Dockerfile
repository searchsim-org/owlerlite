
FROM golang:1.22-alpine AS build
RUN apk add --no-cache git build-base
WORKDIR /src
COPY frontier/proto ./frontier/proto
COPY crawler/ ./crawler
WORKDIR /src/crawler
RUN go mod download
RUN go build -o /out/crawler ./
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=build /out/crawler /crawler
VOLUME ["/data","/warc"]
EXPOSE 8081
ENTRYPOINT ["/crawler"]
