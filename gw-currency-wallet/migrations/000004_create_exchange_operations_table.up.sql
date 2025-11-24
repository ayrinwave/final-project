CREATE TABLE IF NOT EXISTS exchange_operations (
                                                   id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_currency VARCHAR(3) NOT NULL CHECK (from_currency IN ('USD', 'RUB', 'EUR')),
    to_currency VARCHAR(3) NOT NULL CHECK (to_currency IN ('USD', 'RUB', 'EUR')),
    amount BIGINT NOT NULL CHECK (amount > 0),
    exchanged_amount BIGINT NOT NULL CHECK (exchanged_amount > 0),
    rate NUMERIC(20, 10) NOT NULL CHECK (rate > 0),
    request_id TEXT NOT NULL UNIQUE,  -- UNIQUE → индекс автоматически
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    CONSTRAINT check_different_currencies CHECK (from_currency != to_currency)
    );

-- Только нужные индексы (без дублирующего request_id)
CREATE INDEX IF NOT EXISTS idx_exchange_operations_user_id ON exchange_operations(user_id);
CREATE INDEX IF NOT EXISTS idx_exchange_operations_created_at ON exchange_operations(created_at DESC);