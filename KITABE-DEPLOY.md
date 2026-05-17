# Kitabe — scrape.kitabe.org deploy

## Port (sizin bir şey yapmanıza gerek yok)

| Katman | Port |
|--------|------|
| Container içi | **8080** (sabit, `-addr :8080`) |
| Sunucu (host) | **18080** varsayılan (`SCRAPER_HOST_PORT`) |
| nginx | `proxy_pass http://127.0.0.1:18080` |

8080 host’ta başka servis varsa çakışma olmaz. Dış dünya yine `https://scrape.kitabe.org` (443).

## Otomatik deploy (GitHub push)

1. Repo secrets: `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_SSH_KEY`, `DEPLOY_PATH`
2. `main` branch push → `.github/workflows/deploy-kitabe.yml` sunucuda:

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

3. Sunucuda bir kez nginx: `deploy/nginx-scrape.kitabe.org.conf` → sites-enabled

## Manuel (ilk kurulum)

```bash
cd /opt/google-maps-scraper   # DEPLOY_PATH
cp .env.prod.example .env.prod   # isteğe bağlı
docker compose -f docker-compose.prod.yml up -d --build
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:18080/
```

`200` veya `302` beklenir. Sonra: `https://scrape.kitabe.org/api/docs`
