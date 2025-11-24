ALTER TABLE wallets ADD CONSTRAINT balance_non_negative CHECK (balance >= 0);
-- Индекс по request_id НЕ нужен — он уже есть из-за UNIQUE