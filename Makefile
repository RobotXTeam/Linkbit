.PHONY: check test smoke stress recovery-smoke remote-health web-install web-typecheck

GO ?= $(if $(wildcard .tools/go/bin/go),.tools/go/bin/go,go)

check:
	./scripts/check-env.sh

test:
	$(GO) test ./...

smoke:
	./scripts/smoke-api.sh

stress:
	./scripts/stress-api.sh

recovery-smoke:
	./scripts/relay-recovery-smoke.sh

remote-health:
	./scripts/remote-health.sh

web-install:
	cd web && npm install

web-typecheck:
	cd web && npm run typecheck
