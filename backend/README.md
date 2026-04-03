# backend (Phase 1)

Phase 1 hiện tại là backend + CLI thuần cho:
- Keycloak admin API
- OpenVPN Access Server API
- Go REST API + Cobra CLI

Phase này chưa dùng PostgreSQL, outbox hay worker nền.
Email SMTP la optional va duoc gui theo kieu best-effort sau khi nghiep vu thanh cong.

## Quick start

1. Copy env:

```bash
cp .env.example .env
# edit secrets and endpoints
```

2. Build:

```bash
PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOMODCACHE=/tmp/gomodcache GOPATH=/tmp/go GOCACHE=/tmp/gocache go build ./...
```

3. Run server:

```bash
set -a; source .env; set +a
PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOMODCACHE=/tmp/gomodcache GOPATH=/tmp/go GOCACHE=/tmp/gocache go run ./cmd/server
```

4. Use CLI:

```bash
PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOMODCACHE=/tmp/gomodcache GOPATH=/tmp/go GOCACHE=/tmp/gocache go build -o sysctl ./cmd/vpnctl
./sysctl --help
```

## SMTP mail notifications

Mail se duoc gui khi:
- tao user
- enable user
- disable user
- cap nhat VPN access cho user
- cap nhat VPN access cho group (gui den cac member cua group trong Keycloak)

SMTP env:

```bash
SMTP_ENABLED=true
SMTP_HOST=127.0.0.1
SMTP_PORT=1025
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM_ADDRESS=no-reply@example.com
SMTP_FROM_NAME="Hệ thống SSO MBFS"
MAIL_BRAND_NAME="Hệ thống SSO MBFS"
MAIL_SUPPORT_CONTACT="it-support@mobifonesolutions.vn"
MAIL_LOGIN_URL="https://sso.mobifonesolutions.vn/realms/mbfs-solutions/account"
SMTP_TLS_MODE=none
SMTP_TIMEOUT=10s

# Optional: use a separate SMTP account for LDAP account-created emails
SMTP_LDAP_ENABLED=false
SMTP_LDAP_HOST=
SMTP_LDAP_PORT=
SMTP_LDAP_USERNAME=
SMTP_LDAP_PASSWORD=
SMTP_LDAP_FROM_ADDRESS=
SMTP_LDAP_FROM_NAME="Hệ thống SSO MBFS"
SMTP_LDAP_TLS_MODE=
SMTP_LDAP_TIMEOUT=
```

Neu `SMTP_LDAP_*` de trong, sender LDAP se ke thua `SMTP_*`.
Thuong chi can override:

```bash
SMTP_LDAP_ENABLED=true
SMTP_LDAP_USERNAME=...
SMTP_LDAP_PASSWORD=...
SMTP_LDAP_FROM_ADDRESS=...
```

Vi du neu SMTP chinh dang dung Aliyun `465/direct`, sender LDAP se tu ke thua `HOST`, `PORT`, `TLS_MODE`, `TIMEOUT` tu `SMTP_*`, tranh loi auth/TLS do cau hinh lech.

Template hien co:
- `account_created`
- `user_status_changed`
- `vpn_access_changed`

### Test voi Mailpit

Chay Mailpit:

```bash
docker run --rm -p 1025:1025 -p 8025:8025 axllent/mailpit
```

Bat SMTP trong `.env`, restart backend, sau do thu:

```bash
sysctl keycloak user create-local \
  --username test.mail \
  --email test.mail@example.com \
  --first-name "Test" \
  --last-name "Mail" \
  --full-name "Test Mail" \
  --user-type partner \
  --company-name "TEST" \
  --password 'Mbfs@111'

sysctl keycloak user disable --username test.mail

sysctl openvpn access-list append --username test.mail --target 10.0.0.0/8
```

Xem mail tai:

```text
http://127.0.0.1:8025
```

## Keycloak note

- `KEYCLOAK_BASE_URL`: URL admin API, ví dụ `https://admin-sso.mobifonesolutions.vn`
- `KEYCLOAK_TOKEN_URL`: có thể để trống, service sẽ tự tạo từ `KEYCLOAK_BASE_URL`
- Nếu token endpoint ở host khác admin API, set explicit
  ví dụ `https://sso.mobifonesolutions.vn/realms/<realm>/protocol/openid-connect/token`

## Public CLI

- `sysctl keycloak user ...`
- `sysctl keycloak group ...`
- `sysctl openvpn user create-from-keycloak`
- `sysctl openvpn user list`
- `sysctl openvpn group list`
- `sysctl openvpn access-list list`
- `sysctl openvpn access-list append`
- `sysctl openvpn access-list remove`

## Public API

- `POST /api/v1/keycloak/users`
- `GET /api/v1/keycloak/users`
- `GET /api/v1/keycloak/users/{id}`
- `PATCH /api/v1/keycloak/users/{id}`
- `DELETE /api/v1/keycloak/users/{id}`
- `POST /api/v1/keycloak/groups`
- `GET /api/v1/keycloak/groups`
- `GET /api/v1/keycloak/groups/{id}`
- `PATCH /api/v1/keycloak/groups/{id}`
- `DELETE /api/v1/keycloak/groups/{id}`
- `POST /api/v1/keycloak/groups/{id}/members`
- `DELETE /api/v1/keycloak/groups/{id}/members/{userId}`
- `POST /api/v1/openvpn/users:create-from-keycloak`
- `GET /api/v1/openvpn/users`
- `GET /api/v1/openvpn/groups`
- `GET /api/v1/openvpn/access-lists`
- `POST /api/v1/openvpn/access-lists:append`
- `POST /api/v1/openvpn/access-lists:remove`

## Key paths

- `cmd/server/main.go`: backend entrypoint
- `cmd/vpnctl/main.go`: CLI entrypoint
- `internal/application`: application services
- `internal/infrastructure/keycloak`: Keycloak admin client
- `internal/infrastructure/openvpn`: OpenVPN AS API client
- `scripts/post_auth.py`: post-auth policy sample

## Current design note

Phase 1 này ưu tiên tool vận hành trực tiếp qua API/CLI.
OpenVPN `access-list` là giao diện chính cho IP/CIDR và domain:
- IP/CIDR đi qua access list
- domain được backend tự dịch sang rules/ruleset nội bộ của OpenVPN AS
