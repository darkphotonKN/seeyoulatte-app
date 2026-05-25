-- Create ledger_entries table (APPEND-ONLY)
CREATE TABLE ledger_entries (
    id SERIAL PRIMARY KEY,
    order_id UUID REFERENCES orders(id) NOT NULL,
    entry_type VARCHAR(30) NOT NULL,      -- ESCROW, PAYOUT, REFUND, REVERSAL
    amount DECIMAL(10,2) NOT NULL,        -- Always positive. Direction implied by entry_type.
    actor_id UUID,                        -- Who triggered this entry
    actor_type VARCHAR(20),               -- BUYER, SELLER, SYSTEM, ADMIN
    notes TEXT,                           -- Optional context ("Auto-completed after review period")
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT check_ledger_amount CHECK (amount > 0),
    CONSTRAINT check_ledger_entry_type CHECK (entry_type IN ('ESCROW', 'PAYOUT', 'REFUND', 'REVERSAL')),
    CONSTRAINT check_ledger_actor_type CHECK (actor_type IN ('BUYER', 'SELLER', 'SYSTEM', 'ADMIN'))
);

-- Create indexes
CREATE INDEX idx_ledger_entries_order_id ON ledger_entries(order_id);
CREATE INDEX idx_ledger_entries_entry_type ON ledger_entries(entry_type);
CREATE INDEX idx_ledger_entries_actor_id ON ledger_entries(actor_id);
CREATE INDEX idx_ledger_entries_created_at ON ledger_entries(created_at);

-- CRITICAL: Prevent application from mutating ledger rows
-- This would be set for the application user, but for development we'll comment it out
-- REVOKE UPDATE, DELETE ON ledger_entries FROM app_user;