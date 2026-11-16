-- 000004_loyalty.up.sql
CREATE TABLE IF NOT EXISTS loyalty_accounts (
    user_id VARCHAR(64) PRIMARY KEY,
    active_balance INT NOT NULL DEFAULT 0,
    reserved_balance INT NOT NULL DEFAULT 0,
    current_tier VARCHAR(50) NOT NULL DEFAULT 'ROOKIE',
    shows_attended INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS loyalty_reservations (
    id VARCHAR(64) PRIMARY KEY,
    booking_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL REFERENCES loyalty_accounts(user_id) ON DELETE CASCADE,
    reserved_points INT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'RESERVED',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS loyalty_tiers (
    tier_code VARCHAR(50) PRIMARY KEY,
    tier_name VARCHAR(100) NOT NULL,
    min_shows INT NOT NULL,
    cashback_percent INT NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS marketing_rules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    target_genre VARCHAR(100),
    extra_cashback_percent INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Seed tiers
INSERT INTO loyalty_tiers (tier_code, tier_name, min_shows, cashback_percent, description)
VALUES 
('ROOKIE', 'Новичок', 0, 5, '5% кэшбэк баллами на все цирковые билеты.'),
('CONNOISSEUR', 'Знаток манежа', 3, 10, '10% кэшбэк баллами и приоритетная бронь.'),
('HONORARY', 'Почётный зритель', 10, 15, '15% кэшбэк и бесплатные программки.')
ON CONFLICT (tier_code) DO NOTHING;

-- Seed default user for demo: 45a8b9e1-12c4-4b56-8e12-32049e210a55 with 500 points
INSERT INTO loyalty_accounts (user_id, active_balance, reserved_balance, current_tier, shows_attended)
VALUES ('45a8b9e1-12c4-4b56-8e12-32049e210a55', 500, 0, 'CONNOISSEUR', 5)
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO marketing_rules (id, name, target_genre, extra_cashback_percent, is_active)
VALUES (101, 'Двойные баллы на утренние семейные представления', 'Клоунада', 10, TRUE)
ON CONFLICT (id) DO NOTHING;
