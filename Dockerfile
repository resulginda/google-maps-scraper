# Browser + driver cache (same OS family as final debian:trixie-slim)
FROM golang:1.26.2-trixie AS playwright-deps

ENV DEBIAN_FRONTEND=noninteractive
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/browsers

# Pin CLI to the same major line as go.mod (avoid @latest drift vs library).
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates wget \
    && rm -rf /var/lib/apt/lists/* \
    && go install github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 \
    && mkdir -p /opt/browsers \
    && playwright install chromium --with-deps

# Build stage
FROM golang:1.26.2-trixie AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /usr/bin/google-maps-scraper

# Bake Turkey boundaries into the image (avoids runtime TLS timeout to ucdavis.edu).
FROM builder AS geojson-bake
WORKDIR /app
RUN mkdir -p /gmapsdata/geojson/tr/il /gmapsdata/geojson/tr/ilce \
    && (for i in 1 2 3; do \
        CGO_ENABLED=0 go run ./scripts/prepare-turkey-geojson/main.go /gmapsdata && exit 0; \
        echo "geojson bake attempt $i failed, retrying..."; sleep 45; \
    done; echo "geojson bake skipped — container will retry in background")

# Final stage
FROM debian:trixie-slim
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/browsers
ENV PLAYWRIGHT_DRIVER_PATH=/opt

# Install only the necessary dependencies in a single layer
RUN apt-get update && apt-get install -y --no-install-recommends \
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
    && rm -rf /var/lib/apt/lists/*

COPY --from=playwright-deps /opt/browsers /opt/browsers
COPY --from=playwright-deps /root/.cache/ms-playwright-go /opt/ms-playwright-go

RUN chmod -R 755 /opt/browsers \
    && chmod -R 755 /opt/ms-playwright-go

COPY --from=builder /usr/bin/google-maps-scraper /usr/bin/
COPY --from=geojson-bake /gmapsdata/geojson /gmapsdata/geojson

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=90s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/ || exit 1

ENTRYPOINT ["google-maps-scraper"]
# Dokploy Dockerfile build: explicit web mode (no compose command override).
CMD ["-web", "-addr", ":8080", "-data-folder", "/gmapsdata"]
