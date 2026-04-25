.PHONY: check test smoke web-install web-typecheck

GO ?= $(if $(wildcard .tools/go/bin/go),.tools/go/bin/go,go)

check:
	./scripts/check-env.sh

test:
	$(GO) test ./...

smoke:
	./scripts/smoke-api.sh

web-install:
	cd web && npm install

web-typecheck:
	cd web && npm run typecheck
