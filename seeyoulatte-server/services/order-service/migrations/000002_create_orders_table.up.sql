-- Create orders table
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID REFERENCES listings(id) NOT NULL,
    buyer_id UUID REFERENCES users(id) NOT NULL,
    seller_id UUID REFERENCES users(id) NOT NULL,  -- Denormalized from listing for query convenience
    quantity INTEGER NOT NULL DEFAULT 1,            -- How many bags/slots the buyer ordered
    amount DECIMAL(10,2) NOT NULL,                  -- Total price (listing.price * quantity)
    state VARCHAR(30) NOT NULL DEFAULT 'pending_payment',
    seller_respond_by TIMESTAMP,                    -- Deadline for seller to accept/decline
    review_ends_at TIMESTAMP,                       -- Deadline for buyer to dispute after fulfillment
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT check_order_quantity CHECK (quantity > 0),
    CONSTRAINT check_order_amount CHECK (amount > 0),
    CONSTRAINT check_order_state CHECK (state IN (
        'pending_payment', 'paid', 'accepted', 'fulfilled',
        'completed', 'cancelled', 'disputed', 'refunded'
    ))
);

-- Create indexes
CREATE INDEX idx_orders_listing_id ON orders(listing_id);
CREATE INDEX idx_orders_buyer_id ON orders(buyer_id);
CREATE INDEX idx_orders_seller_id ON orders(seller_id);
CREATE INDEX idx_orders_state ON orders(state);
CREATE INDEX idx_orders_seller_respond_by ON orders(seller_respond_by) WHERE seller_respond_by IS NOT NULL;
CREATE INDEX idx_orders_review_ends_at ON orders(review_ends_at) WHERE review_ends_at IS NOT NULL;
CREATE INDEX idx_orders_created_at ON orders(created_at);