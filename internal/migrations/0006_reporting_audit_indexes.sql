CREATE INDEX idx_promotions_product_active ON promotions(product_id, active, created_at);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);
CREATE INDEX idx_expenses_expense_date ON expenses(expense_date);
CREATE INDEX idx_purchase_returns_created_at ON purchase_returns(created_at);
CREATE INDEX idx_sales_returns_created_at ON sales_returns(created_at);
