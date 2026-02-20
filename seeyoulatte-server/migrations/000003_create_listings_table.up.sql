-- Create listings table
CREATE TABLE listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID REFERENCES users(id) NOT NULL,
    title VARCHAR(255) NOT NULL,          -- "Single Origin Ethiopian Yirgacheffe, Light Roast"
    description TEXT,                     -- Details about the coffee, preparation, etc.
    category VARCHAR(20) NOT NULL,        -- 'product' or 'experience'
    price DECIMAL(10,2) NOT NULL,         -- Price per unit (per bag, per seat)
    quantity INTEGER NOT NULL DEFAULT 1,  -- Bags available or experience slots
    pickup_instructions TEXT,             -- "Ring buzzer 3B, 2nd floor" or "Meet at lobby"
    expires_at TIMESTAMP,                 -- Optional. For fresh roasts or limited-time sessions
    is_active BOOLEAN DEFAULT TRUE,       -- Seller can toggle visibility on/off
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT check_quantity CHECK (quantity >= 0),
    CONSTRAINT check_category CHECK (category IN ('product', 'experience'))
);

-- Create indexes
CREATE INDEX idx_listings_seller_id ON listings(seller_id);
CREATE INDEX idx_listings_category ON listings(category);
CREATE INDEX idx_listings_is_active ON listings(is_active);
CREATE INDEX idx_listings_created_at ON listings(created_at);
CREATE INDEX idx_listings_expires_at ON listings(expires_at) WHERE expires_at IS NOT NULL;
