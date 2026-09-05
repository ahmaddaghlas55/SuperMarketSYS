ALTER TABLE dealer_payments ADD COLUMN shift_id INTEGER REFERENCES cash_shifts(id);
ALTER TABLE customer_payments ADD COLUMN shift_id INTEGER REFERENCES cash_shifts(id);
ALTER TABLE expenses ADD COLUMN shift_id INTEGER REFERENCES cash_shifts(id);

CREATE TABLE inventory_owner_expenses (
    id INTEGER PRIMARY KEY,
    product_id INTEGER NOT NULL REFERENCES products(id),
    user_id INTEGER NOT NULL REFERENCES users(id),
    quantity REAL NOT NULL CHECK (quantity > 0),
    unit_cost REAL NOT NULL CHECK (unit_cost >= 0),
    total_value REAL NOT NULL CHECK (total_value >= 0),
    notes TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

CREATE INDEX idx_inventory_owner_expenses_product ON inventory_owner_expenses(product_id);
CREATE INDEX idx_customer_payments_customer ON customer_payments(customer_id);
CREATE INDEX idx_cash_shifts_open ON cash_shifts(closed_at);
