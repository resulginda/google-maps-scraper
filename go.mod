module github.com/gosom/google-maps-scraper

go 1.26.5

require (
	github.com/PuerkitoBio/goquery v1.12.0
	github.com/aws/aws-lambda-go v1.48.0
	github.com/aws/aws-sdk-go-v2 v1.41.5
	github.com/aws/aws-sdk-go-v2/config v1.29.14
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67
	github.com/aws/aws-sdk-go-v2/service/lambda v1.88.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.97.3
	github.com/digitalocean/godo v1.173.0
	github.com/go-chi/chi/v5 v5.2.4
	github.com/golangci/golangci-lint v1.64.8
	github.com/google/open-location-code/go v0.0.0-20250415120251-fa6d7f9d4765
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/gosom/go-leadsdb v0.0.0-20251228094956-ed313efc171f
	github.com/gosom/scrapemate v1.3.0
	github.com/hetznercloud/hcloud-go/v2 v2.36.0
	github.com/jackc/pgx/v5 v5.9.2
	github.com/mattn/go-runewidth v0.0.16
	github.com/mcnijman/go-emailaddress v1.1.1
	github.com/mxschmitt/playwright-go v0.6100.0
	github.com/posthog/posthog-go v1.5.2
	github.com/pquerna/otp v1.5.0
	github.com/riverqueue/river v0.30.1
	github.com/riverqueue/river/riverdriver/riverpgxv5 v0.30.1
	github.com/riverqueue/river/rivertype v0.30.1
	github.com/rubenv/sql-migrate v1.8.1
	github.com/shirou/gopsutil/v4 v4.25.4
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/speps/go-hashids/v2 v2.0.1
	github.com/stretchr/testify v1.11.1
	github.com/swaggo/http-swagger/v2 v2.0.2
	github.com/swaggo/swag v1.16.6
	github.com/urfave/cli/v3 v3.6.2
	golang.org/x/crypto v0.53.0
	golang.org/x/sync v0.21.0
	golang.org/x/term v0.44.0
	modernc.org/sqlite v1.37.0
	riverqueue.com/riverui v0.14.0
)

replace github.com/playwright-community/playwright-go => github.com/mxschmitt/playwright-go v0.6100.0
 
