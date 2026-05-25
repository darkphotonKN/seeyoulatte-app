-- Per-service init migration: ensure UUID extension is available.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
