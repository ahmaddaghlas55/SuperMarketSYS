# Frontend progress

## Fully done

- Shared design tokens, RTL base styles, surfaces, buttons, cards, forms, lists, tables, and responsive layout.
- Shared `api.js` with JSON/FormData requests, auth header attachment, and structured errors.
- Shared `auth.js` with login, logout, token/user storage, and role guards.
- Shared barcode listener, print helpers, `TimelineLedger`, `DataTable`, `NumPad`, `WeightPicker`, `CartLine`, `CustomerPicker`, and `StatusBadge` components.
- Login page with the four seeded Arabic users, all using the typed-password screen. The working `supermarket.db` was updated with bcrypt hashes matching the existing auth scheme: بهاء uses `BA123`; يونس، ابو محمود، and محمود use `1234`. HTTP verification against the running server returned a valid session for all four users.
- Staff home with navigation tiles, user display, logout, and current-shift warning.
- Cash register open/current movement/close flow.
- Dealer receipt creation with dealer/product selection and purchase submission.
- Barcode viewer product lookup and print-label action.
- Admin dashboard summary from `/api/reports/dashboard`.
- Products and categories combined page with product create/edit and category creation.
- Promotions create/edit listing.
- Expenses and expense-category creation.
- Owner inventory expense creation.
- Reports tabs for sales, purchases, stock, and profit summaries.
- Excel exports listing and download links.
- Audit log table.
- Users management route and page removed as requested.

## In progress

- Checkout: cash shortfall now opens an inline searchable customer picker with inline customer creation, preserves the cart, attaches `customer_id`, and submits the shortfall through the documented `amount_received`/server-calculated `remaining` sale flow. Active-promotion labeling, carton-price state, and other previously out-of-scope checkout enhancements remain.
- Debt detail: added pay-this-date, pay-total, and general oldest-first payment actions using the documented `sale_id` payment endpoint. Paid/aging timeline styling remains unchanged.
- Dealer ledger: dealer list, outstanding/credit summary, purchase timeline, and payment form are wired. Payment currently targets the first listed purchase; a purchase-specific selector and richer return-credit timeline remain.
- Purchase returns: single-product return submission is wired. Product-by-product historical return timeline remains.
- Sales returns: debt-removed and cash-refund submission is wired. Exchange item entry/price-difference flow remains.
- All returns: purchase-return timeline is wired. Sales returns need to be combined into the same timeline.
- Staff/admin dealer and category routes intentionally redirect to the combined pages.
- Every page now has a real static HTML layout: headings, stable controls, forms, and named containers are present in the page source; JavaScript only renders data-dependent rows and conditional states. The login avatars/password form and redirect-only pages retain small runtime behavior because those states depend on selection/authentication.

## Not started

- None of the requested page routes remain entirely placeholder-only.

## Decisions

- The four login profiles use the seeded Arabic usernames from the request and always require a typed password. The database-only credential correction used bcrypt exactly as `internal/services/auth.go` does; no backend handlers, services, repositories, or migrations were changed.
- First-run redirects to login because first-time setup was explicitly skipped.
- Google Fonts is linked from each page, with the system Arabic sans stack retained as the CSS fallback.
- No endpoint was added or invented; all requests use paths documented in `API.md`.
