-- 000005_notification.up.sql
CREATE TABLE IF NOT EXISTS notifications (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    channel VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    delivery_status VARCHAR(50) NOT NULL DEFAULT 'DELIVERED',
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id VARCHAR(64) PRIMARY KEY,
    email VARCHAR(255),
    phone VARCHAR(50),
    email_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sms_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    marketing_allowed BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS notification_templates (
    code VARCHAR(100) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    channel VARCHAR(50) NOT NULL,
    body_template TEXT NOT NULL
);

INSERT INTO notification_templates (code, title, channel, body_template)
VALUES
('TICKET_CONFIRMATION', 'Ваши билеты на цирковое представление', 'EMAIL', 'Поздравляем с покупкой! Ваши электронные билеты с QR-кодом во вложении.'),
('BOOKING_EXPIRED', 'Время бронирования истекло', 'EMAIL', 'Здравствуйте! Время оплаты брони истекло. Места вернулись в продажу.'),
('REFUND_PROCESSED', 'Возврат денежных средств', 'EMAIL', 'Ваш возврат 50% стоимости билета успешно проведен.')
ON CONFLICT (code) DO NOTHING;
