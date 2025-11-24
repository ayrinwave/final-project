CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
                                     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
    );

-- Индексы
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Триггер для users (использует универсальную функцию)
CREATE TRIGGER trigger_update_user_timestamp
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Обновление таблицы wallets
ALTER TABLE wallets
    ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    ADD COLUMN user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE;

-- Уникальный индекс: один пользователь — один кошелёк на валюту
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallets_user_currency
    ON wallets(user_id, currency);

-- Ограничение на валюты
ALTER TABLE wallets
    ADD CONSTRAINT check_currency
        CHECK (currency IN ('USD', 'RUB', 'EUR'));

-- Индекс по user_id
CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);

-- Комментарии
COMMENT ON COLUMN wallets.currency IS 'Currency code: USD, RUB, or EUR';
COMMENT ON COLUMN wallets.user_id IS 'Reference to the user who owns this wallet';
COMMENT ON TABLE users IS 'User accounts for the wallet service';