-- Create disputes table
-- NOTE: resolved_by used to reference users(id) but users now lives in
-- auth-service-db, so the FK is dropped here.
CREATE TABLE disputes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES orders(id) NOT NULL,
    reason TEXT NOT NULL,                 -- Buyer's description of the issue
    status VARCHAR(30) DEFAULT 'open',    -- open, resolved_refund, resolved_rejected
    resolved_by UUID,
    resolved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT check_dispute_status CHECK (status IN ('open', 'resolved_refund', 'resolved_rejected'))
);

-- Create indexes
CREATE INDEX idx_disputes_order_id ON disputes(order_id);
CREATE INDEX idx_disputes_status ON disputes(status);
CREATE INDEX idx_disputes_resolved_by ON disputes(resolved_by);
CREATE INDEX idx_disputes_created_at ON disputes(created_at);