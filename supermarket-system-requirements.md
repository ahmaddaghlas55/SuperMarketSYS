# Supermarket Management System — Requirements & Design Reference

**Purpose of this document:** a single source of truth to carry into other chats/agents while building this system. Update it as decisions change — don't let it go stale.

**Context:** Building a POS/inventory/accounting system from scratch for a supermarket that has been running 20+ years. An existing similar system's source code is available for reference, but the explicit intent is to build independently and only use that system afterward as a gap-check against this design — not as a starting point, template, or source to copy/edit.

---

## 1. Environment & Deployment

- **Device:** Samsung Android tablet (a few years old, good quality). Acts as **both server and client** — no separate server machine for now.
- **Runtime:** Termux (Linux userspace on Android) runs the backend continuously.
- **Auto-start:** Termux:Boot launches the backend automatically on power-up, so the system survives power cuts without manual intervention.
- **Networking:** No DNS needed — tablet talks to itself via `localhost`. (Would become relevant if the system later grows to a PC-as-server + multiple tablet-clients setup.)
- **Barcode input:** USB-OTG HID barcode scanner, connected via a USB-C splitter that also passes through charging power (must confirm splitter supports simultaneous charge + OTG data, and that the tablet model supports OTG).
- **Frontend delivery:** Progressive Web App (PWA) — installable to home screen, opens full-screen, works offline via service worker, served from `localhost`.

## 2. Tech Stack & Rationale

| Layer | Choice | Why |
|---|---|---|
| Backend language | **Go** | Existing strength; compiles to a single static binary — no runtime/dependency install needed in Termux |
| Database | **SQLite** | Zero operational overhead, single file, no daemon to keep alive — fits a single-device deployment. Schema written in portable/standard SQL (STRICT tables, avoid SQLite-only syntax, no logic in triggers) so migration to Postgres later is a driver swap, not a rewrite |
| DB access | Go's `database/sql` (generic interface) | Keeps backend code portable across SQL databases |
| Migrations | `golang-migrate` (versioned `.sql` files) | Same migration files can point at SQLite now or Postgres later |
| Frontend | PWA (HTML/CSS/JS + manifest + service worker) | Installed-app feel without app store; works offline against `localhost` |
| UI paradigm | Touch-first | No hover states, no right-click menus — every action is a visible tappable control |
| Barcode | USB-OTG HID scanner (keyboard-emulation) | No custom driver/permission code — reads as keystrokes into a focused input |

**Portability:** Same Go binary cross-compiles to Windows/Linux for a future PC deployment. If the shop grows to multiple devices, migrate DB to Postgres and move to a PC-as-server + tablet-clients architecture (this is also where DNS/local networking would become relevant again).

## 3. UI / Language

- Entire interface in **Arabic**.
- **Checkout/cashier flow** must be extremely simple — usable by anyone, not just technical staff.
- **Admin-side screens** (inventory, dealers, debts, pricing, reports) can be more complex/dense — only the technically capable admin uses these.

## 4. Functional Modules

### 4.1 Sales / Checkout
- Scan barcoded items, or use quick-sell buttons for common non-barcoded items.
- Editable price per line item at time of sale.
- Payment method recorded: cash or card.
- Payment status recorded: paid now, or added to customer's debt.
- **Change calculator:** enter amount customer paid → system shows change due, live. Quick-amount buttons for common bill denominations. If the amount paid is less than the total, the system prompts for a customer name (select existing or create new) and automatically adds the remaining balance to that customer's debt.
- Stock decrements automatically on each sale.

### 4.2 Product Catalog & Non-Barcoded Items
- Full catalog of products, barcoded and non-barcoded.
- **Barcode generator:** assign internal barcodes to items that don't come with one (produce, coffee, cigarette types, etc.) so they become scannable like anything else.
- **Quick-sell buttons** for the most common non-barcoded items, for cashier speed.
- Prices editable at any time by admin.

### 4.3 Inventory
- Tracks quantity on hand per product.
- Decremented by: sales, owner expenses (below), returns to dealer.
- Incremented by: dealer receipts/shipments.
- Design idea: each product has a single **event timeline** (sold / restocked / returned / price-changed / taken-as-expense) rather than separate one-off tables per feature — a per-product history view is just a filtered view of this timeline.

### 4.4 Dealers / Suppliers
- Log a receipt when a dealer delivers stock: items, quantities, **cost price per unit** (asked directly from the dealer at delivery — this is the basis for profit calculation), date.
- Adds delivered quantities to inventory automatically.
- Page showing, per dealer: date, what was bought, running total amount **we owe them**.

### 4.5 Customer Debt
- Per-person page/record: **scrollable UI showing debt by date**.
- Shows purchase history: each date, what was bought, amount, running total owed.
- **Design decision:** never delete purchase records on payoff. Each purchase record gets a `paid` flag/`amount_paid` field (+ `paid_date`). Paid entries are grayed out rather than removed, preserving full history for disputes/audits.
- **Payment granularities:**
  - Pay for **one specific date's** purchase — supports **partial payment** (e.g. owes 50, pays 30, 20 remains outstanding on that record). Needs an `amount_paid` field, not just a boolean.
  - Pay the **total outstanding balance** (general, non-date-specific payment) — applied toward the customer's **oldest debts first**.
- **Aging indicator:** debts unpaid for 1+ month are visually grayed/flagged to draw attention, independent of the paid/grayed-when-paid styling — i.e., two distinct visual states (paid vs. aging-unpaid).
- **Accounting note:** debt payoff affects **neither inventory nor profit** — both were already recorded at the original sale's date (inventory left the shelf then; profit is booked on the sale date regardless of payment status, so a day's profit isn't retroactively changed by later debt collection). Payoff only affects **cash flow** — it's cash entering the register on the day it's paid, relevant to register cash reconciliation (4.11), not the profit report.

### 4.6 Returns (Two Types)

**Sales returns (customer-facing)** — customer made a mistake, wants to exchange or get money back.
- **Bought on debt, returning it:** removes the corresponding entry from their debt record.
- **Paid cash, wants money back:** inventory +1 for the returned product; profit contribution of that original sale line is reversed (uses the cost price already captured at sale time).
- **Exchange for a different item:** modeled as **two linked movements** — a return + a new sale, referencing each other — rather than one combined exchange record. Chosen over a single-record approach specifically for clear per-product return-frequency visibility (a product returned often stands out cleanly in the returns data rather than being buried inside exchange records). Net price difference is what's charged/refunded.

**Purchase returns / مرتجعات (dealer-facing)** — for unsold, expired, or damaged products sent back to the dealer.
- Structurally the mirror of a dealer receipt: instead of stock+debt-to-dealer increasing, stock decreases and it offsets what's owed to the dealer (or generates a credit).
- Fields needed: product, quantity, reason (expired / damaged / unsold), date, effect on dealer balance.
- Feeds into the per-product history timeline (see 4.3) so behavior over time is visible.

### 4.7 Pricing & Units of Sale
- Manual price edit on any item, any time.
- **Units of sale, by product type:**
  - **Produce (fruit/veg) & coffee:** priced by weight — base price per 1kg, system computes price for any weight sold (0.25 / 0.5 / 1 / 2kg, etc.) as `price_per_kg × weight`. Manual override on the line total supported, for basket/marketing deals (e.g. "4kg for 10"). Checkout UI: quick-select buttons for common weights (0.5, 1, 2kg, etc.) instead of a plain +/- stepper; coffee and nuts additionally get smaller increments below 0.5kg (may extend to all weighted items later). **Resolved:** weight entry is manual via these quick-select buttons + numpad — no connected scale in v1, matches how the shop currently operates.
  - **Carton items:** two prices stored — per-piece and per-carton (carton price is normally cheaper). Inventory kept in the smallest unit (pieces), with a "pieces per carton" multiplier. If the quantity of pieces sold in a single sale equals a full carton's worth, the system automatically applies the carton price instead of the per-piece price — no manual unit selection needed by the cashier.
- **Produce discount:** no automated time-based discount curve — cut as it didn't produce the desired behavior. Handled manually instead: the cashier applies a discount directly on the sale line using the existing per-line price override.

### 4.8 Promotions
- Time-boxed or stock-boxed discounts (e.g., 50% off, 2-for-1).
- End condition: either a specified date/time, or "until this item's stock reaches 0."
- **General discount rule:** the newest applied discount/offer on an item overrides any existing one rather than stacking — avoids contradictory pricing (e.g. an active promo plus a separately-discounted price on the same line).

### 4.9 Expenses
**Inventory-based expenses:** when the owner takes stock for personal/house use without paying. Decrements inventory like a sale, but records no revenue. Treated as a loss-of-profit figure, visible only to the admin.

**General operating expenses:** cash costs not tied to a product — electricity, water, etc. Wages deferred/TBD (family-run business, will be specified later). Included in daily profit/net profit calculation.

### 4.10 Reporting
- Daily total profit (sales minus cost price, minus owner expenses).
- Daily expenses figure (admin-only visibility).
- **Later-phase feature:** end-of-day automated service that summarizes the day's sales and emails a written summary to the admin.

### 4.11 Register Cash Tracking
- Opening amount entered at start of day; closing amount entered (counted cash) at end of day.
- System computes expected cash (opening + cash sales − change given − any cash removed) and can surface the difference vs. what was actually counted, to catch shortages/mistakes.

### 4.12 User Hierarchy
- Two roles: **Admin** (full access, plus two extra screens: a day-level dashboard, and a full returns/مرتجعات view) and **Staff/cashier** (sales, debt handling, potentially dealer receipts).
- **Draft permission split (pending confirmation):**
  - **Staff can:** ring up sales, look up/add customer debt, record a dealer receipt, use the change calculator, open/close their own cash shift.
  - **Admin only:** edit prices, create/manage promotions, view profit/reports, manage dealer accounts (owed totals, returns), process customer or dealer returns, manage products/categories, manage users.

### 4.13 Categories & Low-Stock Alerts
- Products organized into categories.
- Each category has a configurable low-stock threshold, set at category creation, used to trigger restock alerts per product.
- **Decided against** formal physical stock-counting/reconciliation for v1 — relying on inventory decrements from sales/expenses/returns only.

### 4.14 Excel Import/Export
- **Product bulk import:** upload a spreadsheet of products instead of entering them one by one; system reads and maps rows into the product catalog. No sample dealer file available yet — expected format (name, barcode, category, unit type, purchase price, sale price) is a starting assumption, to be revisited once a real file is seen.
- **Exports:** system-generated Excel exports for other modules (sales, debt, dealers, returns, expenses, etc.), populated as real data accumulates — for backup/record-keeping and/or external use (e.g., sharing with an accountant). Row granularity (summary vs. per-transaction) to be decided per module as they're built.

## 5. Design Principles Established

- Keep business logic in Go application code, not database triggers/stored procedures (portability).
- Never hard-delete financial history (debt, sales) — use status flags, not deletion, so records remain auditable.
- Prefer simple, explainable logic (e.g., discount curve) over complex fitted models (e.g., regression) unless there's a clear need and enough data to justify it.
- Model recurring shapes (dealer ledger, customer debt ledger, product history) consistently rather than building one-off structures per feature.
- Server always recalculates totals — never trust client-submitted numbers (subtotal, change, etc.) for financial calculations.
- Every multi-step financial/inventory operation runs in a database transaction — no partial writes (e.g., stock decremented but invoice not saved).
- All stock quantity changes must go through a single movement/event record — no code path that changes quantity without logging it (this was a known bug in the reference system's product-edit flow).
- Sale submission should be idempotent (a duplicate/double-tapped request shouldn't create two invoices).

## 5a. Backend Architecture

**Style:** single modular monolith (one Go binary), not microservices — appropriate given everything runs on one tablet against one SQLite file.

**Three internal layers:**
- **API/handlers** — parse and validate HTTP request shape, call services, return JSON. No business logic.
- **Services** — business rules (stock checks, exchange price deltas, shift-close validation, etc.). This is where server-side total recalculation lives.
- **Repository** — the only layer touching SQL, via `database/sql`. Keeps services DB-agnostic, which is what makes a future Postgres migration a repository swap rather than a rewrite.

**Folder structure:**
```
cmd/server/main.go
internal/handlers/    — one file per module (sales, products, debt, dealers, returns, cash, expenses, promotions, reports, users)
internal/services/    — same module split
internal/repository/  — same module split
internal/models/      — shared structs
internal/middleware/  — auth/session + role permission checks
internal/migrations/  — versioned .sql files (golang-migrate)
web/                   — the PWA (index.html, manifest.json, service-worker.js, JS/CSS)
```

**Core data model insight:** most modules reduce to two shared ledgers rather than bespoke logic each:
1. **`stock_movements`** — every inventory change (sale, purchase, return, adjustment, expense) is one row with a `type` column; this is also the per-product history timeline from section 4.3.
2. **`cash_movements`** — every cash-affecting event (cash sale, debt payoff, dealer payment, expense, register open/close) is one row; this feeds register reconciliation (4.11).

**API shape:** plain REST-ish JSON endpoints per module (e.g. `POST /api/sales`, `GET /api/customers/{id}/debt`, `POST /api/customers/{id}/pay`). Every write endpoint runs in a DB transaction with server-side total recalculation.

**Role enforcement:** a single middleware layer checks the logged-in user's role against each route, rather than scattering permission checks inside individual handlers — this is where the admin/staff split (4.12) is actually enforced.

## 6. Open Questions (unresolved as of last update)

1. Exact permission split for the staff/non-admin role — which specific controls beyond the draft list in 4.12 should remain admin-only (this was flagged as "TBD, specifics unspecified").

## 7. Reference System Analysis (Phase 2 — completed)

A full functional report of an existing (unrelated) supermarket system was obtained and compared against this requirements doc, purely as a gap-check — no code or architecture was reviewed or reused. Findings already folded into the sections above:
- Added: customer-facing sales returns (4.6), weight/unit-of-sale pricing model (4.7), general operating expenses (4.9), categories with low-stock thresholds (4.13)
- Confirmed as intentionally different (kept our approach): debt payoff by specific-date-or-total rather than oldest-invoice-first; richer promotions module than the reference system had; explicit damage-reason field on returns
- Adopted as general engineering practice regardless of feature set: see additions to Section 5

## 8. Process Note

Reverse-engineering the existing reference system is planned **only after** this independent design (and eventual implementation) is solid, and only as a narrow gap-check: "did I miss a feature or edge case they handle?" — not as an architecture or code reference. No code or structure from that system is to be copied.

# SS- to run the system : 
```
DATABASE_PATH=/mnt/c/dev/SuperMarketSYS/supermarket.db go run ./cmd/server
```