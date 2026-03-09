---
title: Astrolabe Natal Chart
emoji: ✨
colorFrom: blue
colorTo: indigo
sdk: docker
app_port: 7860
---

# Astrolabe

Go backend + Web UI natal chart app.

## Local Run

```bash
go run ./cmd/server/main.go
```

Open: `http://localhost:8080`

## Deploy on Hugging Face Spaces (Docker)

1. Create a new Space.
2. Select **Docker** SDK.
3. Push this repository files to the Space repo.
4. Space will build and run on port `7860`.

Health check endpoint:

- `/healthz`
