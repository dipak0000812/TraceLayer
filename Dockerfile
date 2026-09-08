FROM golang:1.25 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/tracelayer-api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget
COPY --from=build /out/tracelayer-api /usr/local/bin/tracelayer-api

EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=12 \
    CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/tracelayer-api"]
