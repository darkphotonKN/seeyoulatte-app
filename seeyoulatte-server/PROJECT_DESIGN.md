# SeeYouLatte — System Design & Requirements

**Codename:** SeeYouLatte
**Type:** Weekend POC / Side Project
**Purpose:** Peer-to-peer coffee marketplace. Learn escrow, state machines, race conditions, immutable ledger patterns — all transferable to production marketplace work.

---

## Concept

A peer-to-peer marketplace where people sell coffee to each other. Two modes:

- **Product** — Home-roasted beans, specialty bags, portions from subscription boxes. Buyer orders, picks up.
- **Experience** — "Come to my place, I'll pull you a shot on my espresso setup." Buyer books a slot, shows up.

All transactions are **pickup only**. No delivery. The seller provides pickup instructions and the buyer comes to them.

Items can optionally expire (fresh roasts, limited sessions), but listings are generally persistent — more Airbnb than flash sale.

---

## Core Learning Objectives

| Concept | What We're Learning | Where It Appears |
|---|---|---|
| Race conditions | Multiple buyers, limited quantity. Atomic queries + `FOR UPDATE` locks | Order creation |
| Multi-table business logic | Can't buy if seller frozen, listing expired, insufficient quantity. Cross-table checks need row locks | Order creation |
| State machine | Order flows through defined states with guards and actions. Centralized transition table, no scattered if/else | Order lifecycle |
| Escrow pattern | Platform holds money until fulfillment confirmed + review period ends | Ledger entries tied to state transitions |
| Immutable ledger | Append-only financial records. Never update, never delete. Corrections via reversal entries | `ledger_entries` table |
| Timeout transitions | Background jobs auto-cancel unaccepted orders and auto-complete fulfilled orders when time windows expire | Background worker |

---

## Tech Stack

| Layer | Technology | Notes |
|---|---|---|
| Backend | Go | HTTP server, business logic, background jobs |
| Database | PostgreSQL | Row-level locking, constraints, REVOKE for ledger immutability |
| Frontend | TypeScript + Next.js | Focus is backend; frontend is minimal UI to trigger flows |
| Payments | Mock | No real PSP. Just record ledger entries to simulate money movement |
| Background jobs | Go ticker / cron goroutine | Polls for timed-out orders every minute |
| Auth | JWT | Simple auth with user registration and login |

---

## Database Schema

### Users

Buyers and sellers use the same table. Any user can list coffee or buy coffee.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    bio TEXT,                              -- "Home roaster since 2019, La Marzocca owner..."
    location_text VARCHAR(255),           -- "Da'an District, Taipei"
    is_frozen BOOLEAN DEFAULT FALSE,      -- Admin can freeze bad actors
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Listings

A listing is a coffee product or experience available for purchase.

```sql
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
```

### Orders

An order ties a buyer to a listing with a specific quantity and amount. Tracks state through the lifecycle.

```sql
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
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Ledger Entries (APPEND-ONLY)

Every money movement is an immutable record. Never update. Never delete. Corrections via reversal entries.

```sql
CREATE TABLE ledger_entries (
    id SERIAL PRIMARY KEY,
    order_id UUID REFERENCES orders(id) NOT NULL,
    entry_type VARCHAR(30) NOT NULL,      -- ESCROW, PAYOUT, REFUND, REVERSAL
    amount DECIMAL(10,2) NOT NULL,        -- Always positive. Direction implied by entry_type.
    actor_id UUID,                        -- Who triggered this entry
    actor_type VARCHAR(20),               -- BUYER, SELLER, SYSTEM, ADMIN
    notes TEXT,                           -- Optional context ("Auto-completed after review period")
    created_at TIMESTAMP DEFAULT NOW()
);

-- CRITICAL: Prevent application from mutating ledger rows
REVOKE UPDATE, DELETE ON ledger_entries FROM app_user;
```

### Disputes

Buyer can raise a dispute during the review period. Admin resolves.

```sql
CREATE TABLE disputes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES orders(id) NOT NULL,
    reason TEXT NOT NULL,                 -- Buyer's description of the issue
    status VARCHAR(30) DEFAULT 'open',    -- open, resolved_refund, resolved_rejected
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Reviews

After an order completes, the buyer can leave a review. One review per order.

```sql
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES orders(id) NOT NULL UNIQUE,
    reviewer_id UUID REFERENCES users(id) NOT NULL,
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## Order State Machine

### State Diagram

```
PENDING_PAYMENT
    │
    │  buyer pays (mock)
    ▼
   PAID
    │
    ├── seller accepts ──────────► ACCEPTED
    │                                  │
    │                                  │  seller marks fulfilled
    │                                  │  (buyer picked up / visited)
    │                                  ▼
    │                              FULFILLED
    │                                  │
    │                                  ├── buyer disputes (within 48hr) ──► DISPUTED
    │                                  │                                       │
    │                                  │                                       ├── admin refunds ──► REFUNDED
    │                                  │                                       │
    │                                  │                                       └── admin rejects ──► COMPLETED
    │                                  │
    │                                  │  48hr passes, no dispute (background job)
    │                                  ▼
    │                              COMPLETED ──► payout to seller
    │
    ├── seller declines ─────────► CANCELLED (refund to buyer)
    │
    └── 24hr timeout (bg job) ───► CANCELLED (refund to buyer)
```

### All Possible States

| State | Description |
|---|---|
| `pending_payment` | Order created, awaiting payment |
| `paid` | Payment confirmed, awaiting seller response |
| `accepted` | Seller accepted, awaiting buyer pickup/visit |
| `fulfilled` | Seller confirmed buyer received the coffee. Review period active |
| `completed` | Review period passed or dispute rejected. Seller paid out |
| `cancelled` | Seller declined or response timed out. Buyer refunded |
| `disputed` | Buyer raised issue during review period. Payout frozen |
| `refunded` | Admin resolved dispute in buyer's favor. Buyer refunded |

---

## API Endpoints

### Public

| Method | Path | Description |
|---|---|---|
| GET | `/api/listings` | Browse active listings. Filters: `category`, `search`. Only returns `is_active = true` and `quantity > 0` and not expired |
| GET | `/api/listings/:id` | Listing detail with seller info (name, bio, location, average rating) |

### Auth (Buyer or Seller)

| Method | Path | Description |
|---|---|---|
| GET | `/api/me` | Current user profile |

### Buyer

| Method | Path | Description |
|---|---|---|
| POST | `/api/orders` | Create order. Body: `{ listing_id, quantity }`. Race condition handling here |
| POST | `/api/orders/:id/pay` | Mock payment. Transitions `pending_payment` → `paid` |
| GET | `/api/orders` | My orders as buyer. Query: `?role=buyer` |
| POST | `/api/orders/:id/dispute` | File dispute. Body: `{ reason }`. Only during review period |
| POST | `/api/orders/:id/review` | Leave review. Body: `{ rating, comment }`. Only after `completed` |

### Seller

| Method | Path | Description |
|---|---|---|
| POST | `/api/listings` | Create listing. Body: `{ title, description, category, price, quantity, pickup_instructions, expires_at? }` |
| PATCH | `/api/listings/:id` | Update listing (title, description, price, quantity, is_active, pickup_instructions) |
| GET | `/api/orders?role=seller` | My orders as seller |
| POST | `/api/orders/:id/accept` | Accept order. Transitions `paid` → `accepted` |
| POST | `/api/orders/:id/decline` | Decline order. Transitions `paid` → `cancelled` |
| POST | `/api/orders/:id/fulfill` | Mark fulfilled. Transitions `accepted` → `fulfilled` |

### Admin

| Method | Path | Description |
|---|---|---|
| GET | `/api/admin/disputes` | Open dispute queue |
| POST | `/api/admin/disputes/:id/resolve` | Resolve dispute. Body: `{ resolution: "refund" \| "reject" }` |
| POST | `/api/admin/users/:id/freeze` | Freeze a user |
| POST | `/api/admin/users/:id/unfreeze` | Unfreeze a user |

---

## Build Order (Suggested Sequence)

Each phase builds on the previous.

### Phase 1: Foundation

- [ ] Initialize Go module, install dependencies (`pgx`, `chi`, `uuid`)
- [ ] Set up PostgreSQL Docker Compose
- [ ] Write migration SQL file (full schema from this doc)
- [ ] Database connection pool setup
- [ ] Config loading (env vars)

### Phase 2: Users & Listings CRUD

- [ ] User model + seed data (create a few test users)
- [ ] Listing model + CRUD handlers (create, read, update, list with filters)
- [ ] Simple auth middleware (JWT)
- [ ] Test: can create listings, browse, filter by category

### Phase 3: Order Creation (Race Conditions)

- [ ] Order model
- [ ] `CreateOrder` with `FOR UPDATE` locking (the core learning piece)
- [ ] Mock payment endpoint (`POST /orders/:id/pay`)
- [ ] Ledger entry creation (ESCROW on payment)
- [ ] Test: concurrent order creation (use `go test -race` or a simple load test script)

### Phase 4: State Machine

- [ ] Define transition table (design heart of the system)
- [ ] Implement `TransitionOrder` engine
- [ ] Wire up seller endpoints: accept, decline, fulfill
- [ ] Wire up buyer endpoints: dispute
- [ ] Test: walk an order through the full happy path

### Phase 5: Background Jobs

- [ ] Auto-cancel worker (expired `seller_respond_by`)
- [ ] Auto-complete worker (expired `review_ends_at`)
- [ ] `SKIP LOCKED` + state re-verification pattern
- [ ] Test: create order, don't accept, verify auto-cancellation after timeout

### Phase 6: Disputes & Admin

- [ ] Dispute creation (buyer)
- [ ] Dispute resolution endpoints (admin)
- [ ] Ledger entries on resolution (REFUND or PAYOUT)
- [ ] Admin freeze/unfreeze user
- [ ] Test: full dispute flow

### Phase 7: Reviews

- [ ] Review model + handler
- [ ] Average rating query on listing/seller
- [ ] Test: leave review after completed order

### Phase 8: Frontend

- [ ] Browse listings page
- [ ] Listing detail page
- [ ] Create/edit listing form
- [ ] My orders page (buyer + seller views)
- [ ] Order detail with actions
- [ ] Admin disputes page