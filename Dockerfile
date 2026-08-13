# ============================================================
# 1. PLAYWRIGHT + CHROMIUM
# ============================================================
FROM golang:1.26.2-trixie AS playwright-deps

ENV DEBIAN_FRONTEND=noninteractive
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/browsers
ENV PLAYWRIGHT_DRIVER_PATH=/opt/ms-playwright-go
ENV PLAYWRIGHT_DOWNLOAD_HOST=https://playwright.download.prss.microsoft.com/dbazure/download/playwright

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        wget \
        libnss3 \
        libnspr4 \
        libatk1.0-0 \
        libatk-bridge2.0-0 \
        libcups2 \
        libdrm2 \
        libdbus-1-3 \
        libxkbcommon0 \
        libatspi2.0-0 \
        libx11-6 \
        libxcomposite1 \
        libxdamage1 \
        libxext6 \
        libxfixes3 \
        libxrandr2 \
        libgbm1 \
        libpango-1.0-0 \
        libcairo2 \
        libasound2 \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /opt/browsers /opt/ms-playwright-go \
    && go install github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 \
    && playwright install chromium --with-deps

# ============================================================
# 2. GO BUILD
# ============================================================
FROM golang:1.26.2-trixie AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -o /usr/bin/google-maps-scraper

# ============================================================
# 3. TURKEY GEOJSON
# ============================================================
FROM builder AS geojson-bake

WORKDIR /app

RUN mkdir -p /gmapsdata/geojson/tr/il \
    /gmapsdata/geojson/tr/ilce \
    && (for i in 1 2 3; do \
        CGO_ENABLED=0 go run ./scripts/prepare-turkey-geojson/main.go /gmapsdata \
        && exit 0; \
        echo "geojson bake retry $i/3 in 45s..."; \
        sleep 45; \
    done; \
    echo "geojson bake deferred to container startup")

# ============================================================
# 4. FINAL RUNTIME IMAGE
# ============================================================
FROM debian:trixie-slim

ENV DEBIAN_FRONTEND=noninteractive
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/browsers
ENV PLAYWRIGHT_DRIVER_PATH=/opt/ms-playwright-go
ENV PLAYWRIGHT_DOWNLOAD_HOST=https://playwright.download.prss.microsoft.com/dbazure/download/playwright

# ------------------------------------------------------------
# Chromium runtime dependencies
# ------------------------------------------------------------
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        wget \
        libnss3 \
        libnspr4 \
        libatk1.0-0 \
        libatk-bridge2.0-0 \
        libcups2 \
        libdrm2 \
        libdbus-1-3 \
        libxkbcommon0 \
        libatspi2.0-0 \
        libx11-6 \
        libxcomposite1 \
        libxdamage1 \
        libxext6 \
        libxfixes3 \
        libxrandr2 \
        libgbm1 \
        libpango-1.0-0 \
        libcairo2 \
        libasound2 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    \
    # CRITICAL: verify the library really exists
    && ldconfig \
    && test -e /usr/lib/x86_64-linux-gnu/libxkbcommon.so.0 \
    && ldconfig -p | grep -F "libxkbcommon.so.0"

# ------------------------------------------------------------
# Playwright browser + driver
# ------------------------------------------------------------
COPY --from=playwright-deps /opt/browsers /opt/browsers
COPY --from=playwright-deps /opt/ms-playwright-go /opt/ms-playwright-go

RUN chmod -R 755 /opt/browsers /opt/ms-playwright-go

# ------------------------------------------------------------
# Application
# ------------------------------------------------------------
COPY --from=builder /usr/bin/google-maps-scraper \
    /usr/bin/google-maps-scraper

COPY --from=geojson-bake /gmapsdata/geojson \
    /gmapsdata/geojson

# ------------------------------------------------------------
# Final Chromium sanity check
# ------------------------------------------------------------
RUN test -e /usr/lib/x86_64-linux-gnu/libxkbcommon.so.0 \
    && test -d /opt/browsers \
    && test -d /opt/ms-playwright-go

EXPOSE 8080

HEALTHCHECK \
    --interval=30s \
    --timeout=10s \
    --start-period=90s \
    --retries=3 \
    CMD wget -q -O /dev/null \
        http://127.0.0.1:8080/ || exit 1

ENTRYPOINT ["google-maps-scraper"]

CMD [
    "-web",
    "-addr",
    ":8080",
    "-data-folder",
    "/gmapsdata"
]
