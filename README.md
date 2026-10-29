# Kapufresh ERP API

A small-organization ERP REST API built with Go, Gin, GORM, and pure-Go SQLite.

## Setup

1. Copy `.env.example` to `.env`.
2. Set `ADMIN_EMAIL` and replace `ADMIN_PASSWORD` with a unique password of at least 12 characters. The password is only stored as a bcrypt hash.
3. Replace `JWT_SECRET` with at least 32 bytes of cryptographically random data. Never reuse or commit the secret.
4. Start the API:

   ```powershell
   go run ./cmd/api
   ```

The application creates the SQLite database at `DATABASE_PATH`, runs GORM migrations, and creates the initial admin account only when the users table is empty. Keep `.env` private; it is excluded from version control.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | Application environment |
| `APP_PORT` | `8080` | HTTP port |
| `DATABASE_PATH` | `data/kapufresh.db` | SQLite database file |
| `CURRENCY_CODE` | `KES` | Three-letter currency code for persisted orders |
| `ADMIN_NAME` | `System Administrator` | Initial admin display name |
| `ADMIN_EMAIL` | required | Initial admin login email |
| `ADMIN_PASSWORD` | required | Initial admin password (minimum 12 characters) |
| `JWT_SECRET` | required | Secret for signing access tokens (minimum 32 bytes) |
| `ACCESS_TOKEN_TTL` | `15m` | Access JWT lifetime, as a Go duration |
| `REFRESH_TOKEN_TTL` | `168h` | Refresh token lifetime, as a Go duration |

Access tokens are short-lived JWTs. Refresh tokens are opaque random values; the database stores only their SHA-256 hashes. Refreshing rotates the token, and logout revokes the submitted refresh token. A logged-out access JWT remains valid until it expires, so keep its lifetime short.

## API endpoints

All API routes are versioned under `/api/v1`. Send access tokens as `Authorization: Bearer <access_token>`. Successful responses are wrapped in `data`; errors use `error` with a stable `code` and human-readable `message`.

| Method | Path | Access |
|---|---|---|
| `GET` | `/health` | Public |
| `POST` | `/api/v1/auth/login` | Public |
| `POST` | `/api/v1/auth/refresh` | Public |
| `POST` | `/api/v1/auth/register` | Admin |
| `POST` | `/api/v1/auth/logout` | Authenticated |
| `GET` | `/api/v1/auth/me` | Authenticated |
| `GET` | `/api/v1/users` | Admin |
| `GET` | `/api/v1/users/:id` | Admin |
| `PATCH` | `/api/v1/users/:id` | Admin |
| `DELETE` | `/api/v1/users/:id` | Admin |
| `GET` | `/api/v1/customers` | Authenticated |
| `POST` | `/api/v1/customers` | Admin or manager |
| `GET` | `/api/v1/customers/:id` | Authenticated |
| `PATCH` | `/api/v1/customers/:id` | Admin or manager |
| `DELETE` | `/api/v1/customers/:id` | Admin or manager |
| `GET` | `/api/v1/suppliers` | Authenticated |
| `POST` | `/api/v1/suppliers` | Admin or manager |
| `GET` | `/api/v1/suppliers/:id` | Authenticated |
| `PATCH` | `/api/v1/suppliers/:id` | Admin or manager |
| `DELETE` | `/api/v1/suppliers/:id` | Admin or manager |
| `GET` | `/api/v1/products` | Authenticated |
| `POST` | `/api/v1/products` | Admin or manager |
| `GET` | `/api/v1/products/:id` | Authenticated |
| `PATCH` | `/api/v1/products/:id` | Admin or manager |
| `DELETE` | `/api/v1/products/:id` | Admin or manager |

List routes accept `page`, `per_page` (maximum 100), `search`, `sort`, and `order` (`asc` or `desc`). Products also accept an exact `category` filter. Customer/supplier sort fields are `name`, `email`, and `created_at`; product sort fields are `sku`, `name`, `category`, `unit_price_minor`, `cost_price_minor`, `stock_on_hand`, and `created_at`. Invalid sort fields or directions return `400`.

### Customers

```powershell
curl.exe -H "Authorization: Bearer your-access-token" "http://localhost:8080/api/v1/customers?page=1&per_page=20&search=market&sort=name&order=asc"

curl.exe -X POST http://localhost:8080/api/v1/customers `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"name":"Acme Market","email":"buyer@example.com","phone":"+254700000000","address":"Market Street","notes":"Wholesale customer"}'

curl.exe -H "Authorization: Bearer your-access-token" http://localhost:8080/api/v1/customers/1

curl.exe -X PATCH http://localhost:8080/api/v1/customers/1 `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"phone":"+254711111111","notes":"Updated customer notes"}'

curl.exe -X DELETE http://localhost:8080/api/v1/customers/1 `
  -H "Authorization: Bearer your-manager-or-admin-token"
```

### Suppliers

```powershell
curl.exe -H "Authorization: Bearer your-access-token" "http://localhost:8080/api/v1/suppliers?page=1&per_page=20&search=farm&sort=name&order=asc"

curl.exe -X POST http://localhost:8080/api/v1/suppliers `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"name":"Fresh Farms","email":"sales@example.com","phone":"+254722000000","address":"Farm Road","notes":"Deliveries on weekdays"}'

curl.exe -H "Authorization: Bearer your-access-token" http://localhost:8080/api/v1/suppliers/1

curl.exe -X PATCH http://localhost:8080/api/v1/suppliers/1 `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"address":"New Farm Road"}'

curl.exe -X DELETE http://localhost:8080/api/v1/suppliers/1 `
  -H "Authorization: Bearer your-manager-or-admin-token"
```

### Products

Product prices and costs are integer minor units (for example, `12550` is 125.50 for a two-decimal currency). Product creation does not set opening stock; stock quantity changes belong to the stock-movement endpoints added in Phase 4.

```powershell
curl.exe -H "Authorization: Bearer your-access-token" "http://localhost:8080/api/v1/products?page=1&per_page=20&search=tomato&category=Produce&sort=unit_price_minor&order=asc"

curl.exe -X POST http://localhost:8080/api/v1/products `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"sku":"VEG-001","name":"Tomatoes","category":"Produce","unit":"kg","unit_price_minor":250,"cost_price_minor":140,"reorder_level":5}'

curl.exe -H "Authorization: Bearer your-access-token" http://localhost:8080/api/v1/products/1

curl.exe -X PATCH http://localhost:8080/api/v1/products/1 `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"unit_price_minor":275,"reorder_level":8}'

curl.exe -X DELETE http://localhost:8080/api/v1/products/1 `
  -H "Authorization: Bearer your-manager-or-admin-token"
```

### Health check

```powershell
curl.exe http://localhost:8080/health
```

Expected response:

```json
{"service":"kapufresh-erp","status":"ok"}
```

### Login

```powershell
curl.exe -X POST http://localhost:8080/api/v1/auth/login `
  -H "Content-Type: application/json" `
  -d '{"email":"admin@example.com","password":"your-admin-password"}'
```

Save `access_token` and `refresh_token` from the response.

### Refresh and logout

```powershell
curl.exe -X POST http://localhost:8080/api/v1/auth/refresh `
  -H "Content-Type: application/json" `
  -d '{"refresh_token":"your-refresh-token"}'

curl.exe -X POST http://localhost:8080/api/v1/auth/logout `
  -H "Authorization: Bearer your-access-token" `
  -H "Content-Type: application/json" `
  -d '{"refresh_token":"your-current-refresh-token"}'
```

Refresh tokens rotate. Use the replacement refresh token returned by refresh, and submit that current token to logout.

### Current user and admin user management

```powershell
curl.exe http://localhost:8080/api/v1/auth/me `
  -H "Authorization: Bearer your-access-token"

curl.exe -X POST http://localhost:8080/api/v1/auth/register `
  -H "Authorization: Bearer your-admin-access-token" `
  -H "Content-Type: application/json" `
  -d '{"name":"Store Manager","email":"manager@example.com","password":"a-unique-password-123","role":"manager"}'

curl.exe "http://localhost:8080/api/v1/users?page=1&per_page=20&search=manager&role=manager" `
  -H "Authorization: Bearer your-admin-access-token"

curl.exe http://localhost:8080/api/v1/users/2 `
  -H "Authorization: Bearer your-admin-access-token"

curl.exe -X PATCH http://localhost:8080/api/v1/users/2 `
  -H "Authorization: Bearer your-admin-access-token" `
  -H "Content-Type: application/json" `
  -d '{"name":"Operations Manager","is_active":true}'

curl.exe -X DELETE http://localhost:8080/api/v1/users/2 `
  -H "Authorization: Bearer your-admin-access-token"
```

Registration and user administration are admin-only. The service prevents an admin from disabling or deleting their own account, or removing the final active admin."# kapufresh-erp" 
