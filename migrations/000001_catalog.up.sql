-- 000001_catalog.up.sql
CREATE TABLE IF NOT EXISTS shows (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    genre VARCHAR(100) NOT NULL,
    duration_minutes INT NOT NULL,
    age_restriction INT NOT NULL DEFAULT 0,
    base_price NUMERIC(10, 2) NOT NULL,
    poster_url TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    show_id INT NOT NULL REFERENCES shows(id) ON DELETE CASCADE,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    hall_name VARCHAR(100) NOT NULL DEFAULT 'Главный манеж',
    status VARCHAR(50) NOT NULL DEFAULT 'SCHEDULED'
);

CREATE TABLE IF NOT EXISTS pricing_sectors (
    id SERIAL PRIMARY KEY,
    session_id INT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    sector_code VARCHAR(50) NOT NULL,
    sector_name VARCHAR(100) NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    UNIQUE(session_id, sector_code)
);

-- Тестовые данные
INSERT INTO shows (id, title, description, genre, duration_minutes, age_restriction, base_price, poster_url, status)
VALUES 
(42, 'Магия цирка: Полет над манежем', 'Захватывающее цирковое представление с воздушными гимнастами, акробатами и иллюзионистами.', 'Акробатика и иллюзия', 130, 6, 1200.00, 'https://cdn.circus-theatre.ru/posters/show_42.jpg', 'ACTIVE'),
(43, 'Золотой клоун: Смех и трюки', 'Веселое семейное шоу с клоунами, дрессированными собачками и жонглерами.', 'Клоунада', 110, 0, 1000.00, 'https://cdn.circus-theatre.ru/posters/show_43.jpg', 'ACTIVE')
ON CONFLICT (id) DO NOTHING;

INSERT INTO sessions (id, show_id, start_time, end_time, hall_name, status)
VALUES 
(105, 42, '2026-11-20 18:00:00+00', '2026-11-20 20:10:00+00', 'Главный манеж', 'SCHEDULED'),
(106, 42, '2026-11-21 15:00:00+00', '2026-11-21 17:10:00+00', 'Главный манеж', 'SCHEDULED'),
(107, 43, '2026-11-22 12:00:00+00', '2026-11-22 13:50:00+00', 'Главный манеж', 'SCHEDULED')
ON CONFLICT (id) DO NOTHING;

INSERT INTO pricing_sectors (session_id, sector_code, sector_name, price)
VALUES 
(105, 'PARTERRE_A', 'Партер, сектор А (у манежа)', 2500.00),
(105, 'PARTERRE_B', 'Партер, сектор Б', 2000.00),
(105, 'AMPHITHEATER', 'Амфитеатр', 1500.00),
(105, 'BALCONY', 'Балкон', 1000.00),
(106, 'PARTERRE_A', 'Партер, сектор А (у манежа)', 2500.00),
(106, 'AMPHITHEATER', 'Амфитеатр', 1500.00)
ON CONFLICT (session_id, sector_code) DO NOTHING;
