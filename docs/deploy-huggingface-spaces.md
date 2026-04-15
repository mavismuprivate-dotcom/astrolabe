# Deploy to Hugging Face Spaces

## What is already prepared

- `Dockerfile` builds the Go service and serves the `web/` frontend
- container default port is `7860`
- container default SQLite path is `/app/data/astrolabe.db`
- `.dockerignore` excludes local artifacts from the image build

## Steps

1. Open `https://huggingface.co/new-space`
2. Create a new Space
3. Choose `Docker` as the SDK
4. Push this repository to the Space
5. Wait for the image build to finish

## Verify

- App: `https://<your-space>.hf.space/`
- Health: `https://<your-space>.hf.space/healthz`

Expected health response:

```json
{"status":"ok"}
```

## Persistence note

By default, reports are stored inside the container at `/app/data/astrolabe.db`.
This is enough for preview and testing, but it is not durable across rebuilds.

If you enable persistent storage in Spaces, point `ASTROLABE_DB_PATH` to that mounted path.

