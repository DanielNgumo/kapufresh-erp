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
| `CORS_ALLOWED_ORIGINS` | unset | Comma-separated browser origins allowed to call the API; unset disables cross-origin access |

Access tokens are short-lived JWTs. Refresh tokens are opaque random values; the database stores only their SHA-256 hashes. Refreshing rotates the token, and logout revokes the submitted refresh token. A logged-out access JWT remains valid until it expires, so keep its lifetime short.

For browser clients hosted on another origin, set `CORS_ALLOWED_ORIGINS` to exact origins, without paths or trailing slashes, separated by commas (for example, `http://localhost:5173,https://erp.example.com`). CORS is disabled by default; it does not replace authentication or authorization.

```powershell
curl.exe -i -X OPTIONS http://localhost:8080/api/v1/customers `
  -H "Origin: http://localhost:5173" `
  -H "Access-Control-Request-Method: PATCH" `
  -H "Access-Control-Request-Headers: authorization,content-type"
```

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
| `POST` | `/api/v1/products/:id/stock-movements` | Admin or manager |
| `GET` | `/api/v1/products/:id/stock-movements` | Authenticated |
| `GET` | `/api/v1/stock-movements` | Authenticated |
| `GET` | `/api/v1/inventory/low-stock` | Authenticated |
| `POST` | `/api/v1/sales/orders` | Authenticated |
| `GET` | `/api/v1/sales/orders` | Authenticated |
| `GET` | `/api/v1/sales/orders/:id` | Authenticated |
| `POST` | `/api/v1/sales/orders/:id/confirm` | Authenticated |
| `POST` | `/api/v1/sales/orders/:id/cancel` | Admin or manager |
| `POST` | `/api/v1/sales/orders/:id/payments` | Authenticated |
| `POST` | `/api/v1/purchases/orders` | Admin or manager |
| `GET` | `/api/v1/purchases/orders` | Authenticated |
| `GET` | `/api/v1/purchases/orders/:id` | Authenticated |
| `POST` | `/api/v1/purchases/orders/:id/order` | Admin or manager |
| `POST` | `/api/v1/purchases/orders/:id/receive` | Admin or manager |
| `GET` | `/api/v1/employees` | Authenticated |
| `GET` | `/api/v1/employees/:id` | Authenticated |
| `POST` | `/api/v1/employees` | Admin or manager |
| `PATCH` | `/api/v1/employees/:id` | Admin or manager |
| `DELETE` | `/api/v1/employees/:id` | Admin or manager |
| `GET` | `/api/v1/reports/sales-summary` | Authenticated |
| `GET` | `/api/v1/reports/top-products` | Authenticated |
| `GET` | `/api/v1/reports/stock-valuation` | Authenticated |

List routes accept `page`, `per_page` (maximum 100), `search`, `sort`, and `order` (`asc` or `desc`). Products also accept an exact `category` filter; employees accept `department` and `status`. Customer/supplier sort fields are `name`, `email`, and `created_at`; product sort fields are `sku`, `name`, `category`, `unit_price_minor`, `cost_price_minor`, `stock_on_hand`, and `created_at`; employee sort fields are `first_name`, `last_name`, `email`, `department`, `position`, `status`, and `created_at`. Invalid sort fields or directions return `400`.

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

Product prices and costs are integer minor units (for example, `12550` is 125.50 for a two-decimal currency). Product creation does not set opening stock; add inventory using a stock movement.

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

### Inventory and stock movements

Stock movement requests are transactional: the product balance and its audit history entry either both persist or neither does. Stored movement quantities are signed deltas (`in` positive, `out` negative). For `adjustment`, `quantity` is a signed delta, so a negative value reduces stock; all movements that would make stock negative are rejected. Only admins and managers may record movements. Movement history is newest first.

```powershell
curl.exe -X POST http://localhost:8080/api/v1/products/1/stock-movements `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"type":"in","quantity":25,"reason":"Opening delivery"}'

curl.exe -X POST http://localhost:8080/api/v1/products/1/stock-movements `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"type":"out","quantity":3,"reason":"Damaged goods"}'

curl.exe -X POST http://localhost:8080/api/v1/products/1/stock-movements `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"type":"adjustment","quantity":-2,"reason":"Physical count variance"}'

curl.exe -H "Authorization: Bearer your-access-token" `
  "http://localhost:8080/api/v1/products/1/stock-movements?page=1&per_page=20"

curl.exe -H "Authorization: Bearer your-access-token" `
  "http://localhost:8080/api/v1/stock-movements?product_id=1&type=out&page=1&per_page=20"

curl.exe -H "Authorization: Bearer your-access-token" `
  "http://localhost:8080/api/v1/inventory/low-stock?page=1&per_page=20"
```

Low stock includes products with `stock_on_hand <= reorder_level` (including zero stock); the response includes the shortfall against the reorder level.

### Sales orders and manual payments

Creating an order snapshots current product names, SKUs and prices, and leaves it in `draft`. Confirming deducts all items from stock and writes linked movement rows atomically. If any item lacks stock, the whole confirmation is rejected. Cancelling a confirmed order restores its stock with compensating movements; paid orders cannot be cancelled because refunds are not implemented. Payments are manual (`cash`, `bank`, or `other`); partial payments keep the order confirmed, and a payment covering the full balance marks it paid. Payments cannot exceed the outstanding balance.

```powershell
curl.exe -X POST http://localhost:8080/api/v1/sales/orders `
  -H "Authorization: Bearer your-access-token" `
  -H "Content-Type: application/json" `
  -d '{"customer_id":1,"items":[{"product_id":1,"quantity":2},{"product_id":2,"quantity":1}]}'

curl.exe -H "Authorization: Bearer your-access-token" `
  "http://localhost:8080/api/v1/sales/orders?status=draft&page=1&per_page=20"

curl.exe -H "Authorization: Bearer your-access-token" `
  http://localhost:8080/api/v1/sales/orders/1

curl.exe -X POST http://localhost:8080/api/v1/sales/orders/1/confirm `
  -H "Authorization: Bearer your-access-token"

curl.exe -X POST http://localhost:8080/api/v1/sales/orders/1/payments `
  -H "Authorization: Bearer your-access-token" `
  -H "Content-Type: application/json" `
  -d '{"amount_minor":50000,"method":"cash","reference":"receipt-001"}'

curl.exe -X POST http://localhost:8080/api/v1/sales/orders/1/cancel `
  -H "Authorization: Bearer your-manager-or-admin-token"
```

The payment service defines a `PaymentProvider` interface for a future external provider integration. Phase 5 only records manual payments and does not call any provider.

### Purchase orders and receiving

Purchase orders snapshot item quantities and unit costs, and begin in `draft` without changing stock. Submit a draft order to move it to `ordered`, then receive one or more item quantities at a time. Each receipt increases product stock and writes linked movement rows in a single transaction. Partial receipts remain `ordered`; the order becomes `received` when every line is fully received. Over-receiving is rejected.

```powershell
curl.exe -X POST http://localhost:8080/api/v1/purchases/orders `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"supplier_id":1,"items":[{"product_id":1,"quantity":10,"unit_cost_minor":2500},{"product_id":2,"quantity":5,"unit_cost_minor":4000}]}'

curl.exe -H "Authorization: Bearer your-access-token" `
  "http://localhost:8080/api/v1/purchases/orders?status=draft&page=1&per_page=20"

curl.exe -H "Authorization: Bearer your-access-token" `
  http://localhost:8080/api/v1/purchases/orders/1

curl.exe -X POST http://localhost:8080/api/v1/purchases/orders/1/order `
  -H "Authorization: Bearer your-manager-or-admin-token"

curl.exe -X POST http://localhost:8080/api/v1/purchases/orders/1/receive `
  -H "Authorization: Bearer your-manager-or-admin-token" `
  -H "Content-Type: application/json" `
  -d '{"items":[{"purchase_order_item_id":1,"quantity":4}]}'

```

### Employees

Employee list results support `page`, `per_page`, `search`, `department`, `status`, `sort`, and `order`. Employees can optionally link to a user account; employee deletes are soft deletes. Admins and managers can create, update, and delete employee records.

```powershell
curl.exe -H "Authorization: Bearer <access-token>" `
  "http://localhost:8080/api/v1/employees?page=1&per_page=20&search=amina&department=Sales&status=active&sort=last_name&order=asc"

curl.exe -X POST http://localhost:8080/api/v1/employees `
  -H "Authorization: Bearer <access-token>" `
  -H "Content-Type: application/json" `
  -d '{"first_name":"Amina","last_name":"Otieno","email":"amina@example.com","phone":"+254700000000","department":"Sales","position":"Associate","status":"active"}'

curl.exe -H "Authorization: Bearer <access-token>" http://localhost:8080/api/v1/employees/1

curl.exe -X PATCH http://localhost:8080/api/v1/employees/1 `
  -H "Authorization: Bearer <access-token>" `
  -H "Content-Type: application/json" `
  -d '{"position":"Senior Associate","status":"active"}'

curl.exe -X DELETE http://localhost:8080/api/v1/employees/1 `
  -H "Authorization: Bearer <access-token>"
```

### Reports

All report monetary values are in integer minor units and include the configured `currency_code`. Sales summary and top-product reports require inclusive `start_date` and `end_date` values in `YYYY-MM-DD` format. Sales totals and top products include confirmed and paid orders; payment totals include all recorded receipts by payment date, even if an order was later cancelled. Stock valuation totals cover all active products; its product details are paginated.

```powershell
curl.exe -H "Authorization: Bearer <access-token>" `
  "http://localhost:8080/api/v1/reports/sales-summary?start_date=2025-01-01&end_date=2025-01-31"

curl.exe -H "Authorization: Bearer <access-token>" `
  "http://localhost:8080/api/v1/reports/top-products?start_date=2025-01-01&end_date=2025-01-31&limit=10"

curl.exe -H "Authorization: Bearer <access-token>" `
  "http://localhost:8080/api/v1/reports/stock-valuation?page=1&per_page=20"
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

Registration and user administration are admin-only. The service prevents an admin from disabling or deleting their own account, or removing the final active admin.
