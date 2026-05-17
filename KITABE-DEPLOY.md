# Kitabe — scrape.kitabe.org deploy

## Durum: logda geojson TLS timeout

```
failed to prepare location geojson data: ... gadm41_TUR_1.json: TLS handshake timeout
```

Eski sürümde bu hata **web sunucusunu hiç başlatmıyordu** → Dokploy 502. Güncel sürüm:

- Sunucu **hemen** `:8080`’de açılır
- İl sınırları imaj build sırasında veya arka planda indirilir

Deploy sonrası logda `visit http://localhost:8080` görmelisiniz.

## Durum: container çalışıyor, domain 502 veriyorsa

Deploy logunda `visit http://localhost:8080` görüyorsanız **uygulama ayaktadır**. `Bad Gateway` genelde **Dokploy Domains → container port 8080** eksikliğidir (nginx kullanmıyorsanız sadece bu).

| Katman | Port |
|--------|------|
| Container içi (scraper) | **8080** (`-addr :8080`) |
| Dokploy Domains | Hedef: container **8080** |
| Eski nginx (opsiyonel) | `127.0.0.1:18080` — sadece host’ta bu port publish edildiyse |

---

## Dokploy (sizin kurulum — önerilen)

1. **General** → Repo `resulginda/google-maps-scraper`, branch `main`, Build **Dockerfile** (mevcut ayar doğru).
2. **Domains** sekmesi:
   - Host: `scrape.kitabe.org`
   - **Container port: `8080`** (8080 değilse 502 alırsınız)
   - HTTPS açık
3. **Deploy** veya `main`’e push → autodeploy.
4. Test:
   - `https://scrape.kitabe.org/` → 200 veya 302
   - `https://scrape.kitabe.org/api/docs` → Swagger

### Kalıcı job verisi (isteğe bağlı)

Dokploy → **Mounts** / volumes:

- Container path: `/gmapsdata`
- Named volume veya host path (job CSV’leri burada kalır)

---

## Çakışma: sunucuda ayrı nginx

`deploy/nginx-scrape.kitabe.org.conf` **18080**’e proxy yapar. Dokploy sadece container’ı çalıştırıp **host’ta 18080 publish etmezse** nginx → **502 Bad Gateway**.

**İki seçenekten biri:**

**A) Sadece Dokploy Domains** (kolay)  
- `scrape.kitabe.org` için sunucudaki nginx site’ını kapatın veya DNS’i Dokploy Traefik’e bırakın.  
- Domains → port **8080**.

**B) nginx + host port**  
- Dokploy uygulamasında **Published port** `18080:8080` (veya Advanced → port mapping).  
- nginx `proxy_pass http://127.0.0.1:18080;` aynen kalır.  
- Domains’te aynı domain’i **iki kez** tanımlamayın (nginx ve Dokploy çakışmasın).

Sunucuda kontrol:

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:18080/
# veya Dokploy network üzerinden container IP:8080
```

`200` / `302` beklenir.

---

## docker compose (Dokploy dışı manuel deploy)

```bash
docker compose -f docker-compose.prod.yml up -d --build
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:18080/
```

`.env.prod.example` → `SCRAPER_HOST_PORT=18080` (host 8080 doluysa).

---

## GitHub Actions (opsiyonel)

`.github/workflows/deploy-kitabe.yml` — SSH ile compose deploy. Dokploy autodeploy kullanıyorsanız zorunlu değil.
