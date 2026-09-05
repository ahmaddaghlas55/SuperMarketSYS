-- Supermarket system — initial schema
-- SQLite, written in portable/standard SQL for future Postgres migration.
-- Uses STRICT tables (SQLite feature) for real column typing.
-- No hard deletes on financial/history data — active/paid/void flags instead.

PRAGMA foreign_keys = ON;

-- ─────────────────────────────────────────────
-- Users & roles
-- ─────────────────────────────────────────────

CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('admin', 'staff')),
    active        INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

-- ─────────────────────────────────────────────
-- Categories & products
-- ─────────────────────────────────────────────

CREATE TABLE categories ( --group of diffrent products under the same family
    id                  INTEGER PRIMARY KEY,
    name                TEXT NOT NULL UNIQUE,
    low_stock_threshold INTEGER NOT NULL DEFAULT 5,
    active              INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1))
) STRICT;

-- unit_type drives which pricing columns are meaningful:
--   'piece'  -> sale_price, purchase_price, quantity (whole units)
--   'weight' -> price_per_kg, purchase_price_per_kg, quantity (grams, for precision)
--   'carton' -> price_per_piece, price_per_carton, pieces_per_carton, quantity (pieces)
CREATE TABLE products ( -- one product 
    id                     INTEGER PRIMARY KEY,
    barcode                TEXT UNIQUE,          -- NULL allowed pre-generation; generated barcodes also land here
    name                   TEXT NOT NULL,
    category_id            INTEGER REFERENCES categories(id),
    unit_type              TEXT NOT NULL CHECK (unit_type IN ('piece', 'weight', 'carton')),

    -- piece pricing
    sale_price             REAL,
    purchase_price         REAL,                 -- latest cost, updated on each purchase receipt

    -- weight pricing (produce, coffee, nuts)
    price_per_kg           REAL,
    purchase_price_per_kg  REAL,

    -- carton pricing
    price_per_piece        REAL,
    price_per_carton       REAL,
    pieces_per_carton      INTEGER, 

    quantity               REAL NOT NULL DEFAULT 0,   -- smallest unit for the type (pieces or grams)
    active                 INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at             TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

-- ─────────────────────────────────────────────
-- Dealers / suppliers
-- ─────────────────────────────────────────────

CREATE TABLE dealers (
    id      INTEGER PRIMARY KEY,
    name    TEXT NOT NULL,
    phone   TEXT,
    address TEXT,
    notes   TEXT,
    active  INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1))
) STRICT;

-- A purchase = a dealer receipt/shipment. remaining = what we still owe.
CREATE TABLE purchases (
    id                     INTEGER PRIMARY KEY,
    dealer_id              INTEGER NOT NULL REFERENCES dealers(id),
    user_id                INTEGER NOT NULL REFERENCES users(id),
    dealer_invoice_number  TEXT,
    subtotal               REAL NOT NULL,
    discount               REAL NOT NULL DEFAULT 0,
    total                  REAL NOT NULL,
    paid                   REAL NOT NULL DEFAULT 0,
    remaining              REAL NOT NULL,      -- total - paid; what we owe the dealer
    created_at             TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

CREATE TABLE purchase_items (
    id              INTEGER PRIMARY KEY,
    purchase_id     INTEGER NOT NULL REFERENCES purchases(id),
    product_id      INTEGER NOT NULL REFERENCES products(id),
    quantity        REAL NOT NULL,
    unit_cost       REAL NOT NULL,   -- price per smallest unit at time of purchase
    line_total      REAL NOT NULL --quantity × unit_cost (how much we paied for x product)
) STRICT;

-- A payment we make toward a purchase, reducing what we owe the dealer.
CREATE TABLE dealer_payments (
    id           INTEGER PRIMARY KEY,
    purchase_id  INTEGER NOT NULL REFERENCES purchases(id),
    amount       REAL NOT NULL,
    method       TEXT NOT NULL CHECK (method IN ('cash', 'card', 'transfer')),
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

-- ─────────────────────────────────────────────
-- Purchase returns (مرتجعات — goods sent back to dealer)
-- ─────────────────────────────────────────────

CREATE TABLE purchase_returns (
    id           INTEGER PRIMARY KEY,
    purchase_id  INTEGER NOT NULL REFERENCES purchases(id),
    user_id      INTEGER NOT NULL REFERENCES users(id),
    reason       TEXT NOT NULL CHECK (reason IN ('expired', 'damaged', 'unsold')),
    total_value  REAL NOT NULL,     -- offsets dealer balance first, any excess is a credit/refund
    notes        TEXT,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

CREATE TABLE purchase_return_items (
    id                  INTEGER PRIMARY KEY,
    purchase_return_id  INTEGER NOT NULL REFERENCES purchase_returns(id),
    product_id          INTEGER NOT NULL REFERENCES products(id),
    quantity            REAL NOT NULL,
    unit_cost           REAL NOT NULL,
    line_total          REAL NOT NULL
) STRICT;

-- ─────────────────────────────────────────────
-- Customers & sales (a sale IS the debt record — see amount_paid/remaining)
-- ─────────────────────────────────────────────

CREATE TABLE customers (
    id      INTEGER PRIMARY KEY,
    name    TEXT NOT NULL,
    phone   TEXT,
    active  INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1))
) STRICT;

CREATE TABLE sales (
    id               INTEGER PRIMARY KEY,
    invoice_number   TEXT NOT NULL UNIQUE,
    request_id       TEXT NOT NULL UNIQUE,     -- idempotency key from the POS client
    customer_id      INTEGER REFERENCES customers(id),  -- NULL = walk-in, must be fully paid
    user_id          INTEGER NOT NULL REFERENCES users(id),
    subtotal         REAL NOT NULL,
    discount         REAL NOT NULL DEFAULT 0,
    total            REAL NOT NULL,
    amount_paid      REAL NOT NULL DEFAULT 0,
    remaining        REAL NOT NULL DEFAULT 0,  -- total - amount_paid; the debt on this specific sale/date
    amount_received  REAL,                     -- cash handed over, for change calculation
    change_given     REAL,
    payment_method   TEXT NOT NULL CHECK (payment_method IN ('cash', 'card')),
    created_at       TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

CREATE TABLE sale_items (
    id           INTEGER PRIMARY KEY,
    sale_id      INTEGER NOT NULL REFERENCES sales(id),
    product_id   INTEGER NOT NULL REFERENCES products(id),
    quantity     REAL NOT NULL,       -- pieces, or weight in kg, depending on product.unit_type
    unit_price   REAL NOT NULL,       -- price used for this line (may reflect override/promo)
    cost_price   REAL NOT NULL,       -- product.purchase_price at time of sale — preserved for profit reports
    is_override  INTEGER NOT NULL DEFAULT 0 CHECK (is_override IN (0, 1)),  -- manual price override flag
    line_total   REAL NOT NULL
) STRICT;

-- Debt payments against a specific sale. Multiple partial payments allowed per sale.
CREATE TABLE customer_payments (
    id         INTEGER PRIMARY KEY,
    sale_id    INTEGER NOT NULL REFERENCES sales(id),
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    amount     REAL NOT NULL,
    method     TEXT NOT NULL CHECK (method IN ('cash', 'card')),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

-- ─────────────────────────────────────────────
-- Sales returns (customer-facing: mistake / refund / exchange)
-- ─────────────────────────────────────────────

-- Exchanges are modeled as TWO linked rows: a sales_return + a new sale,
-- connected via linked_sale_id, for clean per-product return-frequency reporting.
CREATE TABLE sales_returns (
    id             INTEGER PRIMARY KEY,
    sale_id        INTEGER NOT NULL REFERENCES sales(id),
    customer_id    INTEGER REFERENCES customers(id),
    user_id        INTEGER NOT NULL REFERENCES users(id),
    refund_type    TEXT NOT NULL CHECK (refund_type IN ('debt_removed', 'cash_refund', 'exchange')),
    linked_sale_id INTEGER REFERENCES sales(id),   -- set only when refund_type = 'exchange'
    total_value    REAL NOT NULL,
    notes          TEXT,
    created_at     TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

CREATE TABLE sales_return_items (
    id                INTEGER PRIMARY KEY,
    sales_return_id   INTEGER NOT NULL REFERENCES sales_returns(id),
    product_id        INTEGER NOT NULL REFERENCES products(id),
    quantity          REAL NOT NULL,
    unit_price        REAL NOT NULL,
    line_total        REAL NOT NULL
) STRICT;

-- ─────────────────────────────────────────────
-- Stock movements — single ledger for every inventory change
-- ─────────────────────────────────────────────

CREATE TABLE stock_movements (
    id               INTEGER PRIMARY KEY,
    product_id       INTEGER NOT NULL REFERENCES products(id),
    type             TEXT NOT NULL CHECK (type IN (
                        'purchase', 'sale', 'return_sale', 'return_purchase',
                        'adjustment', 'owner_expense'
                     )),
    quantity_change  REAL NOT NULL,     -- positive = stock increased, negative = decreased
    balance_after    REAL NOT NULL,
    reference_type   TEXT,              -- e.g. 'sale', 'purchase', 'sales_return', 'purchase_return'
    reference_id     INTEGER,           -- id of the row in that table
    user_id          INTEGER NOT NULL REFERENCES users(id),
    notes            TEXT,
    created_at       TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

-- ─────────────────────────────────────────────
-- General operating expenses (electricity, water, etc. — not inventory-based)
-- ─────────────────────────────────────────────

CREATE TABLE expense_categories (
    id     INTEGER PRIMARY KEY,
    name   TEXT NOT NULL UNIQUE,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1))
) STRICT;

CREATE TABLE expenses (
    id                  INTEGER PRIMARY KEY,
    expense_category_id INTEGER NOT NULL REFERENCES expense_categories(id),
    amount              REAL NOT NULL,
    payment_method      TEXT NOT NULL CHECK (payment_method IN ('cash', 'card', 'bank')),
    expense_date        TEXT NOT NULL,
    notes               TEXT,
    user_id             INTEGER NOT NULL REFERENCES users(id),
    created_at          TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

-- ─────────────────────────────────────────────
-- Cash register / shifts — single ledger for every cash-affecting event
-- ─────────────────────────────────────────────

CREATE TABLE cash_shifts (
    id                INTEGER PRIMARY KEY,
    user_id           INTEGER NOT NULL REFERENCES users(id),
    opening_balance   REAL NOT NULL,
    opening_notes     TEXT,
    opened_at         TEXT NOT NULL DEFAULT (datetime('now')),
    expected_closing  REAL,             -- computed at close time: opening + cash_in - cash_out
    actual_closing    REAL,             -- counted cash, entered at close
    difference        REAL,             -- actual - expected
    closing_notes     TEXT,
    closed_at         TEXT
) STRICT;

CREATE TABLE cash_movements (
    id             INTEGER PRIMARY KEY,
    shift_id       INTEGER NOT NULL REFERENCES cash_shifts(id),
    type           TEXT NOT NULL CHECK (type IN (
                     'sale', 'customer_payment', 'dealer_payment', 'expense',
                     'sales_return_refund', 'purchase_return_refund',
                     'manual_in', 'manual_out'
                   )),
    direction      TEXT NOT NULL CHECK (direction IN ('in', 'out')),
    amount         REAL NOT NULL,
    reference_type TEXT,
    reference_id   INTEGER,
    notes          TEXT,
    created_at     TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

-- ─────────────────────────────────────────────
-- Promotions (time-boxed or stock-boxed offers; override rather than stack)
-- ─────────────────────────────────────────────

CREATE TABLE promotions (
    id               INTEGER PRIMARY KEY,
    product_id       INTEGER NOT NULL REFERENCES products(id),
    promo_type       TEXT NOT NULL CHECK (promo_type IN ('percent_off', 'fixed_price', 'bundle')),
    value            REAL NOT NULL,      -- meaning depends on promo_type (e.g. 50 = 50% off, or a bundle price)
    bundle_quantity  INTEGER,            -- e.g. 2 for "2 for the price of 1"
    starts_at        TEXT,
    ends_at          TEXT,               -- NULL if open-ended (runs until stock hits 0)
    until_stock_zero INTEGER NOT NULL DEFAULT 0 CHECK (until_stock_zero IN (0, 1)),
    active           INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at       TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

-- ─────────────────────────────────────────────
-- Audit log
-- ─────────────────────────────────────────────

CREATE TABLE audit_log (
    id          INTEGER PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    action      TEXT NOT NULL,      -- e.g. 'shift_closed', 'expense_created', 'customer_payment'
    module      TEXT NOT NULL,
    record_id   INTEGER,
    description TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

-- ─────────────────────────────────────────────
-- Indexes for common lookups
-- ─────────────────────────────────────────────

CREATE INDEX idx_products_barcode        ON products(barcode);
CREATE INDEX idx_products_category       ON products(category_id);
CREATE INDEX idx_sales_customer          ON sales(customer_id);
CREATE INDEX idx_sales_created_at        ON sales(created_at);
CREATE INDEX idx_sale_items_sale         ON sale_items(sale_id);
CREATE INDEX idx_stock_movements_product ON stock_movements(product_id);
CREATE INDEX idx_stock_movements_created ON stock_movements(created_at);
CREATE INDEX idx_cash_movements_shift    ON cash_movements(shift_id);
CREATE INDEX idx_purchases_dealer        ON purchases(dealer_id);
CREATE INDEX idx_customer_payments_sale  ON customer_payments(sale_id);
