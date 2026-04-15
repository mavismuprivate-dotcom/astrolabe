# Render Deployment

This repository includes a ready-to-use `render.yaml` for a Render web service.

## What the current blueprint does

- Uses the native Go runtime
- Builds `./cmd/server`
- Starts the app with `./bin/astrolabe`
- Pins `GO_VERSION=1.24.5`
- Stores SQLite data at `/tmp/astrolabe.db`

## Important limitation

`/tmp/astrolabe.db` is writable on Render, but it is not persistent on the free plan.
That means:

- report history can be lost on restart or redeploy
- session-scoped reports are fine for preview/testing
- production persistence needs a Render Disk or an external database

## 1) Push the repository to GitHub

```bash
git add .
git commit -m "chore: prepare deployment config"
git remote add origin <YOUR_GITHUB_REPO_URL>
git branch -M main
git push -u origin main
```

## 2) Create the service from Blueprint

1. Open the Render dashboard.
2. Create a new `Blueprint`.
3. Select the GitHub repository.
4. Confirm the generated `astrolabe` web service.

## 3) Verify the deployment

- App: `https://<your-service>.onrender.com/`
- Health: `https://<your-service>.onrender.com/healthz`

Expected health response:

```json
{"status":"ok"}
```

## 4) If you need persistence later

Move `ASTROLABE_DB_PATH` off `/tmp` and onto persistent storage, or replace SQLite with a managed database.
