-- 000002_booking.up.sql
CREATE TABLE IF NOT EXISTS bookings (
    id VARCHAR(64) PRIMARY KEY,
    session_id INT NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING_PAYMENT',
    total_price NUMERIC(10, 2) NOT NULL,
    points_discount NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    payable_amount NUMERIC(10, 2) NOT NULL,
    payment_url TEXT,
    idempotency_key VARCHAR(64) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS seat_holds (
    id SERIAL PRIMARY KEY,
    session_id INT NOT NULL,
    sector_code VARCHAR(50) NOT NULL,
    row_num INT NOT NULL,
    seat_num INT NOT NULL,
    booking_id VARCHAR(64) NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(session_id, sector_code, row_num, seat_num)
);

CREATE TABLE IF NOT EXISTS tickets (
    ticket_number VARCHAR(64) PRIMARY KEY,
    booking_id VARCHAR(64) NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    session_id INT NOT NULL,
    sector_code VARCHAR(50) NOT NULL,
    row_num INT NOT NULL,
    seat_num INT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    qr_code_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ticket_refunds (
    id VARCHAR(64) PRIMARY KEY,
    booking_id VARCHAR(64) NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL,
    refund_amount NUMERIC(10, 2) NOT NULL,
    penalty_percent INT NOT NULL DEFAULT 50,
    reason TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'REQUESTED',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
