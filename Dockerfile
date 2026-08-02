# Build stage for Playwright dependencies
FROM ubuntu:20.04 AS playwright-deps
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/browsers
ENV PLAYWRIGHT_DRIVER_PATH=/opt/ms-playwright-go
ARG TARGETARCH

# Scrapemate'in zorladığı ve Go'nun aradığı tam sürüm: v0.5700.1 (Driver 1.57.0)
ARG PLAYWRIGHT_GO_VERSION=v0.5700.1
# 404 Hatasını çözen güncel Microsoft CDN adresi
ENV PLAYWRIGHT_DRIVER_URL="https://playwright.download.prss.microsoft.com/dbuyiflzsi96/builds/driver/playwright-%s-%s.zip"

RUN export PATH=$PATH:/usr/local/go/bin:/root/go/bin \
    && apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl wget \
    && if [ "$TARGETARCH" = "arm64" ]; then \
         GO_ARCH="arm64"; \
       else \
         GO_ARCH="amd64"; \
       fi \
    && wget -q "https://go.dev/dl/go1.26.5.linux-${GO_ARCH}.tar.gz" \
    && tar -C /usr/local -xzf "go1.26.5.linux-${GO_ARCH}.tar.gz" \
    && rm "go1.26.5.linux-${GO_ARCH}.tar.gz" \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && go install github.com/playwright-community/playwright-go/cmd/playwright@${PLAYWRIGHT_GO_VERSION} \
    && mkdir -p /opt/browsers \
    && playwright install chromium --with-deps

# Build stage
FROM golang:1.26.5-trixie AS builder
WORKDIR /app
COPY . .
# go.sum imza hatasını ezmek için "tidy" geri eklendi
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /usr/bin/google-maps-scraper

# Bake Turkey boundaries into the image
FROM builder AS geojson-bake
WORKDIR /app
RUN mkdir -p /gmapsdata/geojson/tr/il /gmapsdata/geojson/tr/ilce \
    && (for i in 1 2 3; do \
        CGO_ENABLED=0 go run ./scripts/prepare-turkey-geojson/main.go /gmapsdata && exit 0; \
        echo "geojson bake retry $i/3 in 45s..."; sleep 45; \
    done; echo "geojson bake deferred to container startup")

# Final stage
FROM debian:trixie-slim
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/browsers
ENV PLAYWRIGHT_DRIVER_PATH=/opt/ms-playwright-go
ENV PLAYWRIGHT_DRIVER_URL="https://playwright.download.prss.microsoft.com/dbuyiflzsi96/builds/driver/playwright-%s-%s.zip"

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates wget libnss3 libnspr4 libatk1.0-0 libatk-bridge2.0-0 \
    libcups2 libdrm2 libdbus-1-3 libxkbcommon0 libatspi2.0-0 libx11-6 \
    libxcomposite1 libxdamage1 libxext6 libxfixes3 libxrandr2 libgbm1 \
    libpango-1.0-0 libcairo2 libasound2 \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

COPY --from=playwright-deps /opt/browsers /opt/browsers
COPY --from=playwright-deps /opt/ms-playwright-go /opt/ms-playwright-go

RUN chmod -R 755 /opt/browsers \
    && chmod -R 755 /opt/ms-playwright-go

COPY --from=builder /usr/bin/google-maps-scraper /usr/bin/
COPY --from=geojson-bake /gmapsdata/geojson /gmapsdata/geojson

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=90s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/ || exit 1

ENTRYPOINT ["google-maps-scraper"]
CMD ["-web", "-addr", ":8080", "-data-folder", "/gmapsdata"] 
