-- Internal product barcodes are allocated from a transaction-safe sequence.
CREATE TABLE product_barcode_sequences (
    id         INTEGER PRIMARY KEY CHECK (id = 1),
    next_value INTEGER NOT NULL
) STRICT;

INSERT INTO product_barcode_sequences(id, next_value) VALUES (1, 1);

-- A purchase return first offsets the related purchase. Any excess becomes
-- an unapplied credit that remains available against the dealer balance.
CREATE TABLE dealer_credits (
    id                INTEGER PRIMARY KEY,
    dealer_id         INTEGER NOT NULL REFERENCES dealers(id),
    purchase_return_id INTEGER NOT NULL UNIQUE REFERENCES purchase_returns(id),
    amount            REAL NOT NULL CHECK (amount > 0),
    applied_amount    REAL NOT NULL DEFAULT 0 CHECK (applied_amount >= 0 AND applied_amount <= amount),
    created_at        TEXT NOT NULL DEFAULT (datetime('now'))
) STRICT;

CREATE INDEX idx_dealer_credits_dealer ON dealer_credits(dealer_id);
