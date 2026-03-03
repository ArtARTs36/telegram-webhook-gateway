# telegram-webhook-gateway

telegram-webhook-gateway protects your Telegram bot endpoint from unauthorized access.
How does it protect?

The gateway blocks requests from untrusted IP addresses.

The gateway retrieves trusted IP addresses from the page https://core.telegram.org/resources/cidr.txt. This is done automatically when the process starts, as well as on a schedule: once a day (env `TELEGRAM_CIDR_UPDATE_INTERVAL`)

## Run with docker compose

```yaml
services:
  telegram-webhook-gateway:
    image: artarts36/telegram-webhook-gateway:0.1.0
    environment:
      - TWG_HTTP_ADDR=:8080
      - TWG_TARGET_URL=https://domain.com
      - TWG_IP_HEADERS=X-Real-Ip
```
