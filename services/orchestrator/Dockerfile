
FROM golang:1.22-alpine AS build
RUN apk add --no-cache git build-base
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o /out/orchestrator ./
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=build /out/orchestrator /orchestrator
VOLUME ["/data"]
EXPOSE 7000
ENTRYPOINT ["/orchestrator"]
