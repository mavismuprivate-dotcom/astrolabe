FROM golang:1.24.5-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o /out/astrolabe ./cmd/server

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app && mkdir -p /app/data /app/web && chown -R app:app /app
WORKDIR /app
COPY --from=builder /out/astrolabe /app/astrolabe
COPY web /app/web
ENV PORT=7860
ENV ASTROLABE_DB_PATH=/app/data/astrolabe.db
EXPOSE 7860
USER app
CMD ["/app/astrolabe"]
