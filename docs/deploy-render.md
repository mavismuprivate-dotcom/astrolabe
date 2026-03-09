# Render Free Deployment

This project is ready for Render deployment with the included `render.yaml`.

## 1) Push to GitHub

```bash
git init
git add .
git commit -m "feat: astrolabe natal chart app"
# Replace with your repo
git remote add origin <YOUR_GITHUB_REPO_URL>
git branch -M main
git push -u origin main
```

## 2) Create Render service from blueprint

1. Open Render dashboard.
2. New -> Blueprint.
3. Select this GitHub repo.
4. Render reads `render.yaml` and creates `astrolabe` web service.

## 3) Verify

- Health check: `https://<your-service>.onrender.com/healthz`
- App page: `https://<your-service>.onrender.com/`

## Notes

- Free plan sleeps when idle.
- First request after idle may take longer.
