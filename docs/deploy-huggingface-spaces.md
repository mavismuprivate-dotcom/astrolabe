# Deploy to Hugging Face Spaces (No-card path)

## What is prepared

- `Dockerfile` builds and runs the Go app.
- `README.md` includes Spaces metadata with `sdk: docker` and `app_port: 7860`.

## Steps

1. Open https://huggingface.co/new-space
2. Create a Space (Visibility: public)
3. SDK choose **Docker**
4. In the new Space repo page, upload all project files from this repository
5. Wait for build to complete

## Verify

- App: `https://<your-space>.hf.space/`
- Health: `https://<your-space>.hf.space/healthz`

