# SeeYouLatte — System Design & Requirements

**Codename:** SeeYouLatte
**Type:** Weekend POC / Side Project
**Purpose:** Peer-to-peer coffee marketplace. Covers escrow payment flows, order state machines with guard-based transitions, race condition handling via PostgreSQL row-level locking, append-only immutable ledger for financial audit trails, and background workers for timeout-based state transitions.

---

## Concept

A peer-to-peer marketplace where people sell coffee to each other. Two modes:

- **Product** — Home-roasted beans, specialty bags, portions from subscription boxes. Buyer orders, picks up.
- **Experience** — "Come to my place, I'll pull you a shot on my espresso setup." Buyer books a slot, shows up.

All transactions are **pickup only**. No delivery. The seller provides pickup instructions and the buyer comes to them.

Items can optionally expire (fresh roasts, limited sessions), but listings are generally persistent — more Airbnb than flash sale.

---

## Core Learning Objectives

| Concept                    | What We're Learning                                                                                            | Where It Appears                         |
| -------------------------- | -------------------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| Race conditions            | Multiple buyers, limited quantity. Atomic queries + `FOR UPDATE` locks                                         | Order creation                           |
| Multi-table business logic | Can't buy if seller frozen, listing expired, insufficient quantity. Cross-table checks need row locks          | Order creation                           |
| State machine              | Order flows through defined states with guards and actions. Centralized transition table, no scattered if/else | Order lifecycle                          |
| Escrow pattern             | Platform holds money until fulfillment confirmed + review period ends                                          | Ledger entries tied to state transitions |
| Immutable ledger           | Append-only financial records. Never update, never delete. Corrections via reversal entries                    | `ledger_entries` table                   |
| Timeout transitions        | Background jobs auto-cancel unaccepted orders and auto-complete fulfilled orders when time windows expire      | Background worker                        |

---

## Tech Stack

| Layer           | Technology                 | Notes                                                              |
| --------------- | -------------------------- | ------------------------------------------------------------------ |
| Backend         | Go                         | HTTP server, business logic, background jobs                       |
| Database        | PostgreSQL                 | Row-level locking, constraints, REVOKE for ledger immutability     |
| Frontend        | TypeScript + Next.js       | Focus is backend; frontend is minimal UI to trigger flows          |
| Payments        | Mock                       | No real PSP. Just record ledger entries to simulate money movement |
| Background jobs | Go ticker / cron goroutine | Polls for timed-out orders every minute                            |
| Auth            | JWT                        | Simple auth with user registration and login                       |

---

## User Permissions Model

There are no hard-coded roles like "buyer" or "seller." Permissions are derived from account state. Any authenticated user can buy. A user can sell once they complete seller onboarding.

### Capability Rules

| Action             | Requirement                                                 |
| ------------------ | ----------------------------------------------------------- |
| Browse listings    | None (public, unauthenticated)                              |
| Purchase a listing | Authenticated, not frozen                                   |
| Create a listing   | Authenticated, not frozen, `seller_verified_at IS NOT NULL` |
| Admin actions      | `is_admin = true`                                           |

### Seller Onboarding

A user becomes eligible to sell by completing a seller onboarding step. For this POC, onboarding is a simple endpoint that stamps `seller_verified_at` on the user record. In production, this would be gated behind Stripe Connect account creation, identity verification, or KYC — but the application logic is the same: check whether the timestamp is set.

The `seller_verified_at` field is a timestamp rather than a boolean so there's an audit trail of when onboarding completed. Null means not onboarded. Non-null means eligible to sell.

Mock flow:

1. User calls `POST /api/seller/onboard`
2. System sets `seller_verified_at = NOW()`
3. User can now create listings

Production flow (stretch goal):

1. User initiates Stripe Connect onboarding
2. Stripe webhook confirms connected account is active
3. System sets `seller_verified_at = NOW()` and stores `stripe_account_id`
4. User can now create listings and receive payouts

---

## Database Schema

### Users

Buyers and sellers use the same table. Any authenticated user can buy. Users with completed seller onboarding can also list.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    bio TEXT,                              -- "Home roaster since 2019, La Marzocca owner..."
    location_text VARCHAR(255),           -- "Da'an District, Taipei"
    is_frozen BOOLEAN DEFAULT FALSE,      -- Admin can freeze bad actors
    is_admin BOOLEAN DEFAULT FALSE,       -- Admin privileges
    seller_verified_at TIMESTAMP,         -- Null = can't sell. Non-null = seller onboarding completed
    stripe_account_id VARCHAR(255),       -- Null for mock. Used when real payments added
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

**Notes:**

- `quantity` represents bags for products, slots/seats for experiences.
- `expires_at` is optional. Persistent listings don't need it. Fresh roasts or one-time tasting sessions do.
- `is_active` lets sellers temporarily hide listings without deleting them.
- When `quantity` hits 0, the listing still exists but is not orderable. Frontend shows "Sold out."

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

**Notes:**

- `seller_id` is denormalized from the listing to avoid joins when querying "my orders as a seller."
- `seller_respond_by` is set when order transitions to PAID (`NOW() + 24 hours`).
- `review_ends_at` is set when order transitions to FULFILLED (`NOW() + 48 hours`).

### Ledger Entries (APPEND-ONLY)

Every money movement is an immutable record. Never update. Never delete. Corrections via reversal entries.

```sql
CREATE TABLE ledger_entries (
    id SERIAL PRIMARY KEY,
    order_id UUID REFERENCES orders(id) NOT NULL,
    entry_type VARCHAR(30) NOT NULL,      -- ESCROW, PAYOUT, REFUND, REVERSAL
    amount DECIMAL(10,2) NOT NULL,        -- Always positive. Direction implied by entry_type
    actor_id UUID,                        -- Who triggered this entry
    actor_type VARCHAR(20),               -- BUYER, SELLER, SYSTEM, ADMIN
    notes TEXT,                           -- Optional context ("Auto-completed after review period")
    created_at TIMESTAMP DEFAULT NOW()
);

-- CRITICAL: Prevent application from mutating ledger rows
REVOKE UPDATE, DELETE ON ledger_entries FROM app_user;
```

**Entry types:**

| Entry Type | Meaning                             | When Created                                |
| ---------- | ----------------------------------- | ------------------------------------------- |
| ESCROW     | Buyer's money enters platform hold  | Order transitions to PAID                   |
| PAYOUT     | Platform releases money to seller   | Order transitions to COMPLETED              |
| REFUND     | Platform returns money to buyer     | Order transitions to CANCELLED or REFUNDED  |
| REVERSAL   | Corrects a previous erroneous entry | Manual correction (append negative to undo) |

**Calculating escrow balance for an order:**

```sql
SELECT SUM(
    CASE
        WHEN entry_type IN ('ESCROW') THEN amount
        WHEN entry_type IN ('PAYOUT', 'REFUND', 'REVERSAL') THEN -amount
        ELSE 0
    END
) AS escrow_balance
FROM ledger_entries
WHERE order_id = $1;
-- Result > 0: funds still held. Result = 0: fully disbursed.
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

| State             | Description                                                      |
| ----------------- | ---------------------------------------------------------------- |
| `pending_payment` | Order created, awaiting payment                                  |
| `paid`            | Payment confirmed, awaiting seller response                      |
| `accepted`        | Seller accepted, awaiting buyer pickup/visit                     |
| `fulfilled`       | Seller confirmed buyer received the coffee. Review period active |
| `completed`       | Review period passed or dispute rejected. Seller paid out        |
| `cancelled`       | Seller declined or response timed out. Buyer refunded            |
| `disputed`        | Buyer raised issue during review period. Payout frozen           |
| `refunded`        | Admin resolved dispute in buyer's favor. Buyer refunded          |

### Transition Table

Single source of truth for all allowed state changes. Implement as a centralized transition table in Go.

| #   | From              | To          | Actor  | Guard                                                | Action                                                             |
| --- | ----------------- | ----------- | ------ | ---------------------------------------------------- | ------------------------------------------------------------------ |
| 1   | `pending_payment` | `paid`      | System | Payment confirmed (mock)                             | Create ESCROW ledger entry. Set `seller_respond_by = NOW() + 24hr` |
| 2   | `paid`            | `accepted`  | Seller | Seller not frozen. Within `seller_respond_by` window | —                                                                  |
| 3   | `paid`            | `cancelled` | Seller | Within `seller_respond_by` window                    | Create REFUND ledger entry. Restore listing quantity               |
| 4   | `paid`            | `cancelled` | System | `seller_respond_by` has passed                       | Create REFUND ledger entry. Restore listing quantity               |
| 5   | `accepted`        | `fulfilled` | Seller | —                                                    | Set `review_ends_at = NOW() + 48hr`                                |
| 6   | `fulfilled`       | `disputed`  | Buyer  | `review_ends_at` has NOT passed                      | Freeze payout (no ledger entry yet)                                |
| 7   | `fulfilled`       | `completed` | System | `review_ends_at` has passed AND no open dispute      | Create PAYOUT ledger entry                                         |
| 8   | `disputed`        | `refunded`  | Admin  | —                                                    | Create REFUND ledger entry                                         |
| 9   | `disputed`        | `completed` | Admin  | Dispute rejected                                     | Create PAYOUT ledger entry                                         |

### Quantity Restoration on Cancellation

When an order is cancelled (transitions 3 and 4), the listing quantity must be restored within the same transaction:

```sql
UPDATE listings SET quantity = quantity + $1 WHERE id = $2;
```

---

## Race Condition Handling (Order Creation)

### The Problem

Two buyers try to order the same listing at the same time. Without protection, both read the same quantity, both succeed, and you've oversold.

### The Solution

Lock the listing row AND the seller row within a transaction using `SELECT ... FOR UPDATE OF l, s`. This ensures:

1. Only one transaction reads and acts on the listing at a time
2. Business rule checks (quantity, seller frozen, listing expired) remain valid through the entire transaction
3. The second buyer's SELECT blocks until the first transaction commits or rolls back

### Why Lock Both Tables

The order creation guard checks data from two tables: `listings.quantity`, `listings.expires_at`, and `users.is_frozen`. If you only lock the listing, the seller's `is_frozen` status could change between your read and your write. Locking both rows prevents any mid-transaction mutation.

### Background Jobs and SKIP LOCKED

Background workers that process timed-out orders use `FOR UPDATE SKIP LOCKED` instead of `FOR UPDATE`. This means if another worker already locked a row, this worker skips it instead of blocking. Combined with re-verifying the order state inside each per-row transaction, this prevents double-processing and acting on stale data.

---

## Immutable Ledger Rules

1. **Never UPDATE a ledger row.** Database enforces via `REVOKE UPDATE`.
2. **Never DELETE a ledger row.** Database enforces via `REVOKE DELETE`.
3. **Corrections are new entries.** If an ESCROW amount was wrong, insert a REVERSAL (negative) and a new ESCROW (correct amount).

---

## Background Jobs

Two background jobs run on a ticker (every 60 seconds):

### Job 1: Auto-Cancel Unaccepted Orders

Query: `WHERE state = 'paid' AND seller_respond_by < NOW()`
Action: Transition to `cancelled`, create REFUND ledger entry, restore listing quantity.

### Job 2: Auto-Complete Fulfilled Orders

Query: `WHERE state = 'fulfilled' AND review_ends_at < NOW()` (and no open dispute)
Action: Transition to `completed`, create PAYOUT ledger entry.

Both jobs use `FOR UPDATE SKIP LOCKED` and re-verify state inside each per-row transaction.

---

## API Endpoints

### Public (No Auth)

| Method | Path                 | Description                                                                                                    |
| ------ | -------------------- | -------------------------------------------------------------------------------------------------------------- |
| POST   | `/api/auth/register` | Register new user. Body: `{ email, password, name }`                                                           |
| POST   | `/api/auth/login`    | Login. Returns JWT. Body: `{ email, password }`                                                                |
| GET    | `/api/listings`      | Browse active listings. Filters: `category`, `search`. Returns `is_active = true`, `quantity > 0`, not expired |
| GET    | `/api/listings/:id`  | Listing detail with seller info (name, bio, location, average rating)                                          |

### Authenticated (Any User)

| Method | Path                  | Description                                                   |
| ------ | --------------------- | ------------------------------------------------------------- |
| GET    | `/api/me`             | Current user profile                                          |
| POST   | `/api/seller/onboard` | Complete seller onboarding. Sets `seller_verified_at = NOW()` |

### Buyer (Authenticated, Not Frozen)

| Method | Path                      | Description                                                                  |
| ------ | ------------------------- | ---------------------------------------------------------------------------- |
| POST   | `/api/orders`             | Create order. Body: `{ listing_id, quantity }`. Race condition handling here |
| POST   | `/api/orders/:id/pay`     | Mock payment. Transitions `pending_payment` → `paid`                         |
| GET    | `/api/orders?role=buyer`  | My orders as buyer                                                           |
| POST   | `/api/orders/:id/dispute` | File dispute. Body: `{ reason }`. Only during review period                  |
| POST   | `/api/orders/:id/review`  | Leave review. Body: `{ rating, comment }`. Only after `completed`            |

### Seller (Authenticated, Not Frozen, `seller_verified_at IS NOT NULL`)

| Method | Path                      | Description                                                                                                 |
| ------ | ------------------------- | ----------------------------------------------------------------------------------------------------------- |
| POST   | `/api/listings`           | Create listing. Body: `{ title, description, category, price, quantity, pickup_instructions, expires_at? }` |
| PATCH  | `/api/listings/:id`       | Update listing (title, description, price, quantity, is_active, pickup_instructions)                        |
| GET    | `/api/orders?role=seller` | My orders as seller                                                                                         |
| POST   | `/api/orders/:id/accept`  | Accept order. Transitions `paid` → `accepted`                                                               |
| POST   | `/api/orders/:id/decline` | Decline order. Transitions `paid` → `cancelled`                                                             |
| POST   | `/api/orders/:id/fulfill` | Mark fulfilled. Transitions `accepted` → `fulfilled`                                                        |

### Admin (Authenticated, `is_admin = true`)

| Method | Path                              | Description                                                   |
| ------ | --------------------------------- | ------------------------------------------------------------- |
| GET    | `/api/admin/disputes`             | Open dispute queue                                            |
| POST   | `/api/admin/disputes/:id/resolve` | Resolve dispute. Body: `{ resolution: "refund" \| "reject" }` |
| POST   | `/api/admin/users/:id/freeze`     | Freeze a user                                                 |
| POST   | `/api/admin/users/:id/unfreeze`   | Unfreeze a user                                               |

---

## Pages (Minimal Frontend)

Frontend is minimal. Focus is backend patterns. TypeScript + Next.js.

1. **Browse Listings** (`/`) — Grid of active listings with category filter
2. **Listing Detail** (`/listings/:id`) — Full info, "Buy Now" button, seller rating
3. **Create/Edit Listing** (`/listings/new`, `/listings/:id/edit`) — Form for sellers
4. **My Orders** (`/orders`) — Buyer/seller tabs with state badges and action buttons
5. **Order Detail** (`/orders/:id`) — State timeline, actions, ledger entries, pickup instructions (after accepted)
6. **Admin Disputes** (`/admin/disputes`) — Dispute queue with resolve buttons
7. **Seller Onboarding** (`/seller/onboard`) — Simple page to become a seller

---

## Configuration Constants

Environment variables, easily adjustable:

```go
const (
    SellerResponseTimeout = 24 * time.Hour  // How long seller has to accept/decline
    ReviewPeriod          = 48 * time.Hour  // How long buyer has to dispute after fulfillment
    WorkerInterval        = 1 * time.Minute // How often background jobs check for timeouts
)
```

---

## Build Order

Each phase builds on the previous. Tasks marked 🤖 are good candidates for Claude Code. Unmarked tasks are the core learning pieces to do yourself.

### Phase 1: Foundation

- [ ] 🤖 Initialize Go module, install dependencies (pgx, chi, uuid, jwt)
- [ ] 🤖 PostgreSQL Docker Compose
- [ ] 🤖 Migration SQL file (full schema from this doc)
- [ ] 🤖 Database connection pool setup
- [ ] 🤖 Config loading (env vars)

### Phase 2: Auth & Users

- [ ] 🤖 User model + registration/login handlers
- [ ] 🤖 JWT middleware
- [ ] 🤖 Seller onboarding endpoint
- [ ] 🤖 Seed data (test users: one regular, one verified seller, one admin)

### Phase 3: Listings CRUD

- [ ] 🤖 Listing model + CRUD handlers with seller_verified_at guard
- [ ] 🤖 List with filters (category, active, not expired)
- [ ] Test: verified seller can create listing, unverified user cannot

### Phase 4: Order Creation (Race Conditions) ← First thing you code yourself

- [ ] Order model
- [ ] `CreateOrder` with `FOR UPDATE` locking
- [ ] Mock payment endpoint
- [ ] Ledger entry creation (ESCROW on payment)
- [ ] Test: concurrent order creation with goroutines

### Phase 5: State Machine

- [ ] Define transition table
- [ ] Implement `TransitionOrder` engine
- [ ] Wire up seller endpoints: accept, decline, fulfill
- [ ] Wire up buyer endpoints: dispute
- [ ] Test: full happy path

### Phase 6: Background Jobs

- [ ] Auto-cancel worker (expired `seller_respond_by`)
- [ ] Auto-complete worker (expired `review_ends_at`)
- [ ] `SKIP LOCKED` + state re-verification pattern
- [ ] Test: verify auto-cancellation and auto-completion

### Phase 7: Disputes & Admin

- [ ] Dispute creation (buyer)
- [ ] 🤖 Dispute resolution endpoints (admin)
- [ ] Ledger entries on resolution
- [ ] 🤖 Admin freeze/unfreeze user

### Phase 8: Reviews

- [ ] 🤖 Review model + handler
- [ ] 🤖 Average rating query

### Phase 9: Frontend

- [ ] 🤖 All pages (browse, detail, orders, admin, onboarding)

---

## Key Design Decisions

**Derived permissions over roles:** No "buyer" or "seller" role column. Capabilities are computed from account state (seller_verified_at, is_frozen, is_admin). This is how real marketplaces work — Airbnb doesn't have a "host role," you become a host by completing onboarding.

**seller_verified_at as timestamp not boolean:** Provides audit trail of when onboarding completed. Null means not onboarded.

**seller_id denormalized on orders:** Avoids joins through listings for "my orders as seller" queries.

**Mock payments:** Payment integration (Stripe) is a project in itself. Escrow/ledger patterns are identical whether mock or real. Can swap in Stripe later without changing state machine or ledger.

**REVOKE UPDATE/DELETE on ledger:** Defense in depth. Even with application bugs, the database physically prevents ledger mutation.

**Re-verify state in background jobs:** Between batch SELECT and per-row processing, state could change. Re-checking inside each transaction prevents acting on stale data.

---

## Stretch Goals (Post-Weekend)

- [ ] Real payment integration (Stripe Connect for seller payouts)
- [ ] Image uploads for listings (S3)
- [ ] WebSocket notifications (order state changes)
- [ ] Seller ratings aggregate + badge system
- [ ] Search with location/distance (PostGIS)
- [ ] Rate limiting on order creation
- [ ] Real KYC via Stripe Identity for seller onboarding
