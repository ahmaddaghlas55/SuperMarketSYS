# SuperMarket API

The server listens on `:8080` by default. JSON requests require
`Content-Type: application/json`. Authenticated examples use
`Authorization: Bearer $TOKEN`; replace `$TOKEN` with the token returned by
login. Examples use `http://localhost:8080`.
`admin` means administrator only; `staff` means either staff or admin.

## Public and authentication

| Method/path | Auth | Body | Response | curl |
|---|---|---|---|---|
| `GET /healthz` | none | — | `{"status":"ok"}` | `curl http://localhost:8080/healthz` |
| `POST /api/auth/bootstrap` | none, first installation only | `{"username":"admin","password":"至少8chars"}` | `201 {"user":{...}}` | `curl -X POST -H 'Content-Type: application/json' -d '{"username":"admin","password":"change-me"}' http://localhost:8080/api/auth/bootstrap` |
| `POST /api/auth/login` | none | `{"username":"...","password":"..."}` | `{"token","expires_at","user"}` | `curl -X POST -H 'Content-Type: application/json' -d '{"username":"admin","password":"change-me"}' http://localhost:8080/api/auth/login` |
| `GET /api/auth/me` | staff | — | `{"user":{...}}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/auth/me` |
| `POST /api/auth/logout` | staff | — | `{"ok":true}` | `curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/auth/logout` |

## Users

| Method/path | Auth | Body | Response | curl |
|---|---|---|---|---|
| `GET /api/users` | admin | — | `{"users":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/users` |
| `POST /api/users` | admin | `{"username","password","role":"admin\|staff"}` | `201 user` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"username":"cashier","password":"password8","role":"staff"}' http://localhost:8080/api/users` |
| `PUT /api/users/{id}` | admin | `{"username","role","password,omitempty"}` | `{"ok":true}` | `curl -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"username":"cashier","role":"staff"}' http://localhost:8080/api/users/2` |
| `DELETE /api/users/{id}` | admin | — | `{"ok":true,"active":false}` | `curl -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/users/2` |
| `POST /api/users/{id}/activate` | admin | — | `{"ok":true,"active":true}` | `curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/users/2/activate` |

## Categories and products

| Method/path | Auth | Body | Response | curl |
|---|---|---|---|---|
| `GET /api/categories` | staff | — | `{"categories":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/categories` |
| `POST /api/categories` | admin | `{"name","low_stock_threshold"}` | `201 category` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Produce","low_stock_threshold":5}' http://localhost:8080/api/categories` |
| `PUT /api/categories/{id}` | admin | category fields | `{"ok":true}` | `curl -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Produce","low_stock_threshold":8}' http://localhost:8080/api/categories/1` |
| `DELETE /api/categories/{id}` | admin | — | `{"ok":true}` (soft delete) | `curl -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/categories/1` |
| `POST /api/categories/{id}/activate` | admin | — | `{"ok":true,"active":true}` | `curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/categories/1/activate` |
| `GET /api/products?low_stock=true` | staff | — | `{"products":[...]}` | `curl -H "Authorization: Bearer $TOKEN" 'http://localhost:8080/api/products?low_stock=true'` |
| `GET /api/products/{id}` | staff | — | product | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/products/1` |
| `GET /api/products/{id}/price?quantity=2` | staff | — | `{"product_id","quantity","unit_price","line_total"}` | `curl -H "Authorization: Bearer $TOKEN" 'http://localhost:8080/api/products/1/price?quantity=2'` |
| `POST /api/products` | admin | Product fields (`name`, `unit_type`, pricing; optional `barcode`, `quantity`) | `201 {"product", "barcode_generated"}` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Milk","unit_type":"piece","sale_price":2.5}' http://localhost:8080/api/products` |
| `PUT /api/products/{id}` | admin | Product fields | `200 product` | `curl -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Milk","unit_type":"piece","sale_price":2.7}' http://localhost:8080/api/products/1` |
| `DELETE /api/products/{id}` | admin | — | `{"ok":true}` (soft delete) | `curl -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/products/1` |
| `POST /api/products/{id}/activate` | admin | — | `{"ok":true,"active":true}` | `curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/products/1/activate` |
| `GET /api/products/{id}/barcode-label` | staff | — | PDF barcode label | `curl -H "Authorization: Bearer $TOKEN" -o label.pdf http://localhost:8080/api/products/1/barcode-label` |
| `POST /api/products/import` | admin | multipart field `file` (Excel) | `{"imported":n}` | `curl -X POST -H "Authorization: Bearer $TOKEN" -F file=@products.xlsx http://localhost:8080/api/products/import` |

## Promotions

Promotion types are `percent_off`, `fixed_price`, and `bundle`. A promotion is
selected newest-first; promotions never stack. A sale line's explicit
`price_override`/`line_total` always wins.

| Method/path | Auth | Body | Response | curl |
|---|---|---|---|---|
| `GET /api/promotions?product_id=&active=false` | staff | — | `{"promotions":[...]}`; default is currently active only | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/promotions` |
| `GET /api/promotions/active?product_id=` | staff | — | `{"promotions":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/promotions/active` |
| `GET /api/promotions/{id}` | staff | — | promotion | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/promotions/1` |
| `POST /api/promotions` | admin | `{"product_id","promo_type","value","bundle_quantity,omitempty","starts_at,omitempty","ends_at,omitempty","until_stock_zero":true}` | `201 promotion` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"product_id":1,"promo_type":"percent_off","value":20}' http://localhost:8080/api/promotions` |
| `PUT /api/promotions/{id}` | admin | promotion fields | `{"ok":true}` | `curl -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"product_id":1,"promo_type":"fixed_price","value":1.99}' http://localhost:8080/api/promotions/1` |
| `DELETE /api/promotions/{id}` | admin | — | `{"ok":true}` (soft delete) | `curl -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/promotions/1` |
| `POST /api/promotions/{id}/activate` | admin | — | `{"ok":true,"active":true}` | `curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/promotions/1/activate` |

## Sales, customers, debt, and returns

Sale item fields are `product_id`, `quantity`, and optional `price_override`
or `line_total`. The server recalculates all totals, cost, promotions, stock,
change, and debt. `request_id` makes sale submission idempotent.

| Method/path | Auth | Body | Response | curl |
|---|---|---|---|---|
| `POST /api/sales` | staff | `{"request_id","customer_id,omitempty","discount":0,"amount_received,omitempty","amount_paid,omitempty","payment_method":"cash\|card","items":[...]}` | `201 sale` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"request_id":"pos-1","payment_method":"cash","amount_received":10,"items":[{"product_id":1,"quantity":2}]}' http://localhost:8080/api/sales` |
| `GET /api/sales?customer_id=&date=YYYY-MM-DD&payment_status=paid\|unpaid\|partial` | staff | — | `{"sales":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/sales` |
| `GET /api/sales/{id}` | staff | — | sale with items | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/sales/1` |
| `POST /api/sales/{id}/return` | admin | `{"refund_type":"debt_removed\|cash_refund\|exchange","customer_id,omitempty","items":[...],"exchange_items,omitempty","notes,omitempty"}` | `201 sales return` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"refund_type":"cash_refund","items":[{"product_id":1,"quantity":1}]}' http://localhost:8080/api/sales/1/return` |
| `GET /api/customers` | staff | — | `{"customers":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/customers` |
| `POST /api/customers` | staff | `{"name","phone,omitempty"}` | `201 customer` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Ali"}' http://localhost:8080/api/customers` |
| `GET /api/customers/{id}` | staff | — | customer with sales debt | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/customers/1` |
| `PUT /api/customers/{id}` | admin | `{"name","phone,omitempty"}` | `{"ok":true}` | `curl -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Ali","phone":"555"}' http://localhost:8080/api/customers/1` |
| `DELETE /api/customers/{id}` | admin | — | `{"ok":true}` (soft delete) | `curl -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/customers/1` |
| `POST /api/customers/{id}/activate` | admin | — | `{"ok":true,"active":true}` | `curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/customers/1/activate` |
| `GET /api/customers/{id}/debt` | staff | — | `{"customer", "sales":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/customers/1/debt` |
| `POST /api/customers/{id}/payments` | staff | `{"sale_id":0,"amount","method":"cash\|card"}` | `{"customer_id","applied_amount"}` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"amount":20,"method":"cash"}' http://localhost:8080/api/customers/1/payments` |
| `POST /api/customers/{id}/pay` | staff | same as payments | same | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"sale_id":3,"amount":5,"method":"card"}' http://localhost:8080/api/customers/1/pay` |
| `POST /api/sales-returns` | admin | same return body as sale return, including `sale_id` | `201 sales return` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"sale_id":1,"refund_type":"debt_removed","items":[{"product_id":1,"quantity":1}]}' http://localhost:8080/api/sales-returns` |

## Dealers, purchases, and purchase returns

| Method/path | Auth | Body | Response | curl |
|---|---|---|---|---|
| `GET /api/dealers` | staff | — | `{"dealers":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/dealers` |
| `POST /api/dealers` | admin | `{"name","phone,omitempty","address,omitempty","notes,omitempty"}` | `201 dealer` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Supplier"}' http://localhost:8080/api/dealers` |
| `PUT /api/dealers/{id}` | admin | dealer fields | `{"ok":true}` | `curl -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Supplier 2"}' http://localhost:8080/api/dealers/1` |
| `DELETE /api/dealers/{id}` | admin | — | `{"ok":true}` (soft delete) | `curl -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/dealers/1` |
| `POST /api/dealers/{id}/activate` | admin | — | `{"ok":true,"active":true}` | `curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/dealers/1/activate` |
| `GET /api/purchases?dealer_id=` | staff | — | `{"purchases":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/purchases` |
| `GET /api/purchases/{id}` | staff | — | purchase with items | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/purchases/1` |
| `POST /api/purchases` | staff | `{"dealer_id","dealer_invoice_number,omitempty","discount":0,"items":[{"product_id","quantity","unit_cost"}]}` | `201 purchase` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"dealer_id":1,"items":[{"product_id":1,"quantity":10,"unit_cost":1.2}]}' http://localhost:8080/api/purchases` |
| `POST /api/purchases/{id}/payments` | admin | `{"amount","method":"cash\|card\|transfer"}` | purchase | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"amount":10,"method":"cash"}' http://localhost:8080/api/purchases/1/payments` |
| `POST /api/purchase-returns` | admin | purchase return with `purchase_id`, `reason:"expired\|damaged\|unsold"`, `items:[{"product_id","quantity"}]`, optional `notes` | `201 purchase return` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"purchase_id":1,"reason":"damaged","items":[{"product_id":1,"quantity":1}]}' http://localhost:8080/api/purchase-returns` |
| `GET /api/purchase-returns?purchase_id=` | admin | — | `{"returns":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/purchase-returns` |

## Cash and expenses

| Method/path | Auth | Body | Response | curl |
|---|---|---|---|---|
| `GET /api/cash-shifts` | staff | — | `{"shifts":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/cash-shifts` |
| `GET /api/cash/shifts/current` | staff | — | current shift | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/cash/shifts/current` |
| `GET /api/cash-shifts/{id}/movements` | staff | — | `{"movements":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/cash-shifts/1/movements` |
| `POST /api/cash-shifts` or `POST /api/cash/shifts/open` | staff | `{"opening_balance","opening_notes,omitempty"}` | `201 shift` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"opening_balance":100}' http://localhost:8080/api/cash-shifts` |
| `POST /api/cash-shifts/{id}/close` | staff (owner only) | `{"actual_closing","closing_notes,omitempty"}` | closed shift | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"actual_closing":125}' http://localhost:8080/api/cash-shifts/1/close` |
| `POST /api/cash/movements` | staff | `{"direction":"in\|out","amount","notes,omitempty"}` | `201 cash movement` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"direction":"out","amount":10,"notes":"drawer removal"}' http://localhost:8080/api/cash/movements` |
| `GET /api/expense-categories` | staff | — | `{"categories":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/expense-categories` |
| `POST /api/expense-categories` | admin | `{"name"}` | `201 category` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Utilities"}' http://localhost:8080/api/expense-categories` |
| `PUT /api/expense-categories/{id}` | admin | `{"name"}` | `{"ok":true}` | `curl -X PUT -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"name":"Electricity"}' http://localhost:8080/api/expense-categories/1` |
| `DELETE /api/expense-categories/{id}` | admin | — | `{"ok":true}` (soft delete) | `curl -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/expense-categories/1` |
| `POST /api/expense-categories/{id}/activate` | admin | — | `{"ok":true,"active":true}` | `curl -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/expense-categories/1/activate` |
| `POST /api/expenses` | admin | `{"expense_category_id","amount","payment_method":"cash\|card\|bank","expense_date,omitempty","notes,omitempty"}` | `201 expense` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"expense_category_id":1,"amount":50,"payment_method":"cash"}' http://localhost:8080/api/expenses` |
| `POST /api/inventory-owner-expenses` | admin | `{"product_id","quantity","notes,omitempty"}` | `201 inventory owner expense` | `curl -X POST -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"product_id":1,"quantity":2}' http://localhost:8080/api/inventory-owner-expenses` |

## Reports, exports, and audit

Date filters use `from=YYYY-MM-DD&to=YYYY-MM-DD`.

| Method/path | Auth | Body | Response | curl |
|---|---|---|---|---|
| `GET /api/reports/sales` | admin | — | `{"invoice_count","gross_sales","discounts_given","returns_total","net_sales","average_invoice","paid_total","remaining_total"}` | `curl -H "Authorization: Bearer $TOKEN" 'http://localhost:8080/api/reports/sales?from=2026-01-01&to=2026-01-31'` |
| `GET /api/reports/purchases` | admin | — | `{"purchase_count","gross_purchases","purchase_returns","net_purchases","paid_total","dealer_balances":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/reports/purchases` |
| `GET /api/reports/stock` | admin | — | `{"product_count","total_stock_value","low_stock_products":[...],"out_of_stock_products":[...]}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/reports/stock` |
| `GET /api/reports/profit` | admin | — | `{"net_sales","cost_of_goods_sold","gross_profit","general_expenses","net_profit","loss_of_profit"}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/reports/profit` |
| `GET /api/reports/dashboard` | admin | — | `{"today_sales_total","today_profit","low_stock_count","open_shift","today_expenses","today_returns_count"}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/reports/dashboard` |
| `GET /api/exports/sales.xlsx` | admin | — | Excel workbook download | `curl -H "Authorization: Bearer $TOKEN" -o sales.xlsx http://localhost:8080/api/exports/sales.xlsx` |
| `GET /api/exports/debt.xlsx` | admin | — | Excel workbook download | `curl -H "Authorization: Bearer $TOKEN" -o debt.xlsx http://localhost:8080/api/exports/debt.xlsx` |
| `GET /api/exports/dealers-purchases.xlsx` | admin | — | Excel workbook download | `curl -H "Authorization: Bearer $TOKEN" -o purchases.xlsx http://localhost:8080/api/exports/dealers-purchases.xlsx` |
| `GET /api/exports/expenses.xlsx` | admin | — | Excel workbook download | `curl -H "Authorization: Bearer $TOKEN" -o expenses.xlsx http://localhost:8080/api/exports/expenses.xlsx` |
| `GET /api/exports/returns.xlsx` | admin | — | Excel workbook download | `curl -H "Authorization: Bearer $TOKEN" -o returns.xlsx http://localhost:8080/api/exports/returns.xlsx` |
| `GET /api/audit-log?module=&from=&to=&user_id=` | admin | — | `{"audit":[...]}` | `curl -H "Authorization: Bearer $TOKEN" 'http://localhost:8080/api/audit-log?module=cash'` |
| `ANY /api/admin/ping` | admin | — | `{"ok":true,"role":"admin"}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/admin/ping` |
| `ANY /api/staff/ping` | staff | — | `{"ok":true}` | `curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/staff/ping` |

The report response shapes are summary objects (not row arrays): sales includes
`invoice_count`, `gross_sales`, `discounts_given`, `returns_total`,
`net_sales`, `average_invoice`, `paid_total`, and `remaining_total`;
purchases includes `purchase_count`, `gross_purchases`, `purchase_returns`,
`net_purchases`, `paid_total`, and `dealer_balances`; stock includes
`product_count`, `total_stock_value`, `low_stock_products`, and
`out_of_stock_products`; profit includes `net_sales`, `cost_of_goods_sold`,
`gross_profit`, `general_expenses`, `net_profit`, and `loss_of_profit`;
dashboard includes `today_sales_total`, `today_profit`, `low_stock_count`,
`open_shift`, `today_expenses`, and `today_returns_count`. The audit endpoint
also accepts `module`.

Errors use `{"error":{"code":"..."}}`; common codes include
`authentication_required`, `invalid_session`, `insufficient_role`,
`invalid_request`, `not_found`, and `conflict`.
