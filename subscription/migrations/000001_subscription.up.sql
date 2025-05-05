CREATE TABLE IF NOT EXISTS currencies (
    code CHAR(3) PRIMARY KEY,            -- ISO 4217 currency code
    name VARCHAR(50) NOT NULL
);

-- Insert currencies
INSERT INTO currencies (code, name)
VALUES
    ('USD', 'US Dollar'),
    ('IDR', 'Indonesian Rupiah')
ON CONFLICT (code) DO NOTHING;  -- Prevents duplicates if currencies already exist

-- Plans
CREATE TABLE IF NOT EXISTS plans (
    id             SERIAL          PRIMARY KEY,
    name           VARCHAR(255)    NOT NULL,
    description    TEXT,
    price          NUMERIC(10,2)   NOT NULL,
    currency       CHAR(3)         NOT NULL REFERENCES currencies(code),
    duration_days  INT             NOT NULL CHECK (duration_days > 0) DEFAULT 30,
    created_at     TIMESTAMP       NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP       NOT NULL DEFAULT NOW(),
    version        INT             NOT NULL DEFAULT 1
);

-- Insert $2 USD plan
INSERT INTO plans (name, description, price, currency, duration_days)
VALUES
    ('Basic Plan', 'Entry-level subscription', 19000.00, 'IDR', 30),
    ('Premium Plan', 'Mid-tier subscription', 39000.00, 'IDR', 30);

-- Discounts (coupons)
CREATE TABLE IF NOT EXISTS discounts (
    id                   SERIAL          PRIMARY KEY,
    code                 VARCHAR(50)     NOT NULL UNIQUE,
    description          TEXT,
    scope                VARCHAR(20)     NOT NULL CHECK (scope IN ('ALL','PLANS')) DEFAULT 'ALL',
    percentage_value     NUMERIC(10,2)   NOT NULL,
    start_date           TIMESTAMP       NOT NULL DEFAULT NOW(),
    end_date             TIMESTAMP,
    created_at           TIMESTAMP       NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP       NOT NULL DEFAULT NOW(),
    version              INT             NOT NULL DEFAULT 1
);

-- Discounts plans
CREATE TABLE IF NOT EXISTS discount_plans (
    discount_id INT NOT NULL REFERENCES discounts(id) ON DELETE CASCADE,
    plan_id     INT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    PRIMARY KEY (discount_id, plan_id)
);

SELECT * FROM discounts WHERE code = 'JS200';
-- Return if discount exists and scope is 'ALL'
-- Else check in the discount_plans table
SELECT d.* FROM discounts d JOIN discount_plans dp on d.id = dp.discount_id WHERE dp.plan_id = 2 AND code = 'JS200';

-- I'll use this one for getting valid discounts
SELECT d.*
FROM discounts d
WHERE d.code = 'JS200' AND (d.scope = 'ALL'
    OR (d.scope = 'PLANS' AND EXISTS (
            SELECT 1
            FROM discount_plans dp
            WHERE dp.discount_id = d.id AND dp.plan_id = 2
        )
    )
);

-- Orders (one per purchase intent)
CREATE TABLE IF NOT EXISTS orders (
    id         UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    INT       NOT NULL REFERENCES users(id),
    plan_id    INT       NOT NULL REFERENCES plans(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Transactions (payments) link back to orders
CREATE TABLE IF NOT EXISTS transactions (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id         UUID           NOT NULL REFERENCES orders(id),
    subtotal         NUMERIC(10,2)  NOT NULL, -- plan price before fees/taxes
    tax              NUMERIC(10,2),  -- calculated tax
    processing_fee   NUMERIC(10,2),  -- Stripe/PayPal/etc. fee passed to user
    discount         NUMERIC(10,2),  -- promotional or coupon value
    total            NUMERIC(10,2)  NOT NULL,
    currency         CHAR(3)        NOT NULL REFERENCES currencies(code),
    status           VARCHAR(20)    NOT NULL CHECK (status IN ('PENDING','COMPLETED','FAILED')),
    idempotency_key  UUID           NOT NULL UNIQUE,
    created_at       TIMESTAMP      NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP      NOT NULL DEFAULT NOW(),
    version          INT            NOT NULL DEFAULT 1
);

-- Subscriptions with pause/resume metadata
CREATE TABLE IF NOT EXISTS subscriptions (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        INT         NOT NULL REFERENCES users(id),
    plan_id        INT         NOT NULL REFERENCES plans(id),
    start_date     TIMESTAMP   NOT NULL DEFAULT NOW(),
    end_date       TIMESTAMP,                     -- set by trigger
    status         VARCHAR(20) NOT NULL CHECK (status IN ('ACTIVE','PAUSED','EXPIRED')) DEFAULT 'ACTIVE',
    paused_at      TIMESTAMP,                     -- when it was paused
    remaining_days INT,                           -- days left upon pause
    created_at     TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP   NOT NULL DEFAULT NOW(),
    version        INT         NOT NULL DEFAULT 1,
    deleted_at     TIMESTAMP
);

-- Partial unique index: only one ACTIVE subscription per user
CREATE UNIQUE INDEX ux_user_active
    ON subscriptions(user_id)
    WHERE status = 'ACTIVE';

-- Webhook logging for payment gateway callbacks
CREATE TABLE IF NOT EXISTS payment_events (
    id           UUID     PRIMARY KEY DEFAULT gen_random_uuid(),
    tx_id        UUID     NOT NULL REFERENCES transactions(id),
    payload      JSONB    NOT NULL,
    received_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    status       VARCHAR(20) NOT NULL DEFAULT 'NOTIFIED'
);

CREATE OR REPLACE FUNCTION set_purchase_end_date()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.end_date := NEW.start_date + (
        SELECT duration_days * INTERVAL '1 DAY'
        FROM plans
        WHERE id = NEW.plan_id
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_purchase_end_date_trigger
    BEFORE INSERT ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION set_purchase_end_date();

-- 9. Example pg_cron job: expire old ACTIVES and resume oldest PAUSED nightly.
-- Requires: `CREATE EXTENSION IF NOT EXISTS pg_cron;`
-- SELECT cron.schedule(
--     'expire_and_resume_subscriptions',
--     '0 0 * * *',  -- every day at midnight UTC
--     $$
--
--     -- Expire ACTIVE subscriptions past end_date
--     UPDATE subscriptions
--     SET status = 'EXPIRED'
--     WHERE status = 'ACTIVE' AND end_date < NOW();
--
--     -- Resume oldest PAUSED per user
--     WITH to_resume AS (
--         SELECT DISTINCT ON (user_id) id
--         FROM subscriptions
--         WHERE status = 'PAUSED'
--         ORDER BY user_id, paused_at ASC
--     )
--     UPDATE subscriptions s
--     SET
--         start_date   = NOW(),
--         end_date     = NOW() + (s.remaining_days * INTERVAL '1 DAY'),
--         status       = 'ACTIVE'
--     FROM to_resume tr
--     WHERE s.id = tr.id;
--
--     $$
-- );
