-- Create users table for authentication and marketplace
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255), -- NULL for OAuth-only users
    name VARCHAR(255) NOT NULL,
    bio TEXT,
    location_text VARCHAR(255),
    is_frozen BOOLEAN DEFAULT FALSE,

    -- OAuth fields
    google_id VARCHAR(255) UNIQUE,
    avatar_url VARCHAR(500),

    -- Marketplace defaults
    is_verified BOOLEAN DEFAULT FALSE,
    preferred_pickup_instructions TEXT,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_google_id ON users(google_id);
CREATE INDEX idx_users_created_at ON users(created_at);

-- Add updated_at trigger
CREATE TRIGGER set_users_timestamp
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();