CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    provider VARCHAR(20) NOT NULL CHECK (provider IN ('GOOGLE', 'FACEBOOK', 'GITHUB')) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL, -- Unique ID from the provider (e.g., Google's 'sub')
    picture VARCHAR(255) DEFAULT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    username VARCHAR(255) DEFAULT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    version integer NOT NULL DEFAULT 1,
    UNIQUE (provider, provider_user_id)
);

CREATE INDEX IF NOT EXISTS idx_users_provider_user_id ON users (provider_user_id);

-- our own token, not from providers
CREATE TABLE IF NOT EXISTS tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    access_token VARCHAR(255) UNIQUE NOT NULL,
    refresh_token BYTEA UNIQUE DEFAULT NULL,
    scopes VARCHAR(1024) NOT NULL,
    access_token_expires_at TIMESTAMP NOT NULL,
    refresh_token_expires_at TIMESTAMP DEFAULT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP DEFAULT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
