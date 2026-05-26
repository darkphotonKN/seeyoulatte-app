# Workspace-level Makefile for SeeYouLatte microservices.
#
# Per-service Makefiles live under services/*/Makefile and are the source of
# truth for that service's commands. This file fans those out across the
# workspace and adds a verify-builds target that asserts build artifacts land
# only in the expected places (bin/ and tmp/, both gitignored).

SERVICES := api-gateway auth-service baseline-service order-service

.PHONY: build-all clean-all verify-builds tidy-all docker-up docker-down

# Build every service's binary via its own `make build`.
build-all:
	@for svc in $(SERVICES); do \
		echo "=== build $$svc ==="; \
		$(MAKE) -C services/$$svc build || exit 1; \
	done

# Remove bin/ and tmp/ from every service.
clean-all:
	@for svc in $(SERVICES); do \
		echo "=== clean $$svc ==="; \
		$(MAKE) -C services/$$svc clean || true; \
	done
	@rm -rf ./tmp ./bin

# Run go mod tidy in every module (common/ + every service).
tidy-all:
	@cd common && go mod tidy
	@for svc in $(SERVICES); do \
		echo "=== tidy $$svc ==="; \
		(cd services/$$svc && go mod tidy) || exit 1; \
	done

# Build every service AND assert each binary landed at services/<name>/bin/<name>.
# Then scan the tree for any compiled Go binary outside the allowed locations
# (services/*/bin/ and services/*/tmp/) so a misconfigured Makefile or .air.toml
# can't dump artifacts somewhere git would pick up.
verify-builds: clean-all build-all
	@echo ""
	@echo "=== asserting binaries are at services/<name>/bin/<name> ==="
	@fail=0; for svc in $(SERVICES); do \
		path="services/$$svc/bin/$$svc"; \
		if [ ! -x "$$path" ]; then \
			echo "  MISSING: $$path"; fail=1; \
		else \
			echo "  OK:      $$path"; \
		fi; \
	done; \
	if [ $$fail -ne 0 ]; then echo "verify-builds FAILED: missing binaries"; exit 1; fi
	@echo ""
	@echo "=== scanning for stray binaries outside bin/ + tmp/ ==="
	@stray=$$(find . -type f -perm +111 \
		-not -path "./.git/*" \
		-not -path "*/bin/*" \
		-not -path "*/tmp/*" \
		-not -path "*/node_modules/*" \
		-not -path "*/scripts/*" \
		-not -name "*.sh" -not -name "*.go" -not -name "*.md" \
		-not -name "*.sql" -not -name "*.yml" -not -name "*.yaml" \
		-not -name "*.toml" -not -name "*.mod" -not -name "*.sum" \
		-not -name "Makefile" -not -name ".env*" -not -name ".gitignore" \
		-not -name ".air.toml" -not -name "*.proto" -not -name "*.pb.go" \
		-not -name "*.png" -not -name "*.jpg" \
		2>/dev/null \
		| xargs -I{} sh -c 'file "{}" | grep -qE "Mach-O|ELF|executable" && echo "{}"' 2>/dev/null \
	); \
	if [ -n "$$stray" ]; then \
		echo "STRAY binaries detected:"; \
		echo "$$stray"; \
		echo "verify-builds FAILED: rebuild paths in Makefiles / .air.toml"; \
		exit 1; \
	else \
		echo "  none found"; \
	fi
	@echo ""
	@echo "=== checking .gitignore would exclude built artifacts ==="
	@cd $(CURDIR) && git check-ignore services/api-gateway/bin/api-gateway >/dev/null 2>&1 \
		&& echo "  OK:      services/api-gateway/bin/* is gitignored" \
		|| { echo "  FAIL:    services/api-gateway/bin/* NOT gitignored"; exit 1; }
	@cd $(CURDIR) && git check-ignore services/auth-service/tmp/main >/dev/null 2>&1 \
		&& echo "  OK:      services/auth-service/tmp/* is gitignored" \
		|| { echo "  FAIL:    services/auth-service/tmp/* NOT gitignored"; exit 1; }
	@echo ""
	@echo "verify-builds OK"

# Bring up the root infrastructure (Consul, RabbitMQ, Redis, per-service Postgres).
docker-up:
	@docker compose up -d

docker-down:
	@docker compose down
