# DDD API Template

A production-ready Go API following Domain-Driven Design principles with clean architecture.

## Quick Start

```bash
# Start infrastructure
make docker-up

# Run migrations
make migrate-up

# Start development server
make dev
```

Your API will be running at `http://localhost:8080`

## Available Commands

```bash
make dev          # Run with hot reload (air)
make build        # Build binary
make test         # Run tests
make lint         # Run linter
make docker-up    # Start PostgreSQL & Redis
make docker-down  # Stop containers
make migrate-up   # Run migrations
make migrate-down # Rollback migrations
```

## API Endpoints

### Item Management

- `GET /api/items` - List all items
- `POST /api/items` - Create new item
- `GET /api/items/{id}` - Get specific item
- `PUT /api/items/{id}` - Update item
- `DELETE /api/items/{id}` - Delete item

### Health Check
- `GET /health` - API health check

## Environment Variables

Copy `.env.example` to `.env` and configure as needed.

## Development

This project follows Domain-Driven Design (DDD) principles with clean architecture. See `CLAUDE.md` for detailed development guidelines and patterns.

## Testing

```bash
make test        # All tests
make test-item   # Item domain tests
```

## Git Workflow

- Commit messages: `type: description` (feat, fix, test, refactor, chore, docs)
- Never commit code that fails `make lint && make test`

## License

MIT