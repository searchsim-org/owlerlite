
FROM golang:1.22-alpine AS build
RUN apk add --no-cache git build-base
WORKDIR /src/frontier
COPY frontier/ .
RUN go mod download
RUN go build -o /out/frontier ./
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /
COPY --from=build /out/frontier /frontier
VOLUME ["/data"]
EXPOSE 7071 7072
ENTRYPOINT ["/frontier"]
