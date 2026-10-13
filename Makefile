.PHONY: check test build backlog packet issue-drafts migration-create migration-seal migration-check migration-test ledger-test web-check web-test web-build web-browser app-integration

check:
	go run ./cmd/project check
	go run ./cmd/migrate -action=verify
	bash scripts/migrate.sh check
	bash scripts/check-stage-runbook.sh --validate
	go vet ./...
	git diff --check

test:
	go test -race ./...
	bash scripts/test-migration-files.sh

build:
	go build -trimpath -o bin/project ./cmd/project
	go build -trimpath -o bin/corpus ./cmd/corpus
	go build -trimpath -o bin/api ./cmd/api
	go build -trimpath -o bin/migrate ./cmd/migrate

web-check:
	npm --prefix web run api:check
	npm --prefix web run check

web-test:
	npm --prefix web test

web-build:
	npm --prefix web run build

web-browser:
	npm --prefix web run test:browser

app-integration:
	bash scripts/test-app-integration.sh

.PHONY: demo demo-test
demo:
	bash scripts/test-app-integration.sh --demo

demo-test:
	bash scripts/test-app-integration.sh --demo-test

backlog:
	go run ./cmd/project backlog

packet:
	go run ./cmd/project packet $(ID)

issue-drafts:
	go run ./cmd/project drafts

migration-create:
	bash scripts/migrate.sh create "$(NAME)"

migration-seal:
	bash scripts/migrate.sh seal

migration-check:
	bash scripts/migrate.sh check

migration-test:
	bash scripts/migrate.sh test

ledger-test:
	bash scripts/test-ledger.sh

.PHONY: infra-test
infra-test:
	bash scripts/test-infra.sh

.PHONY: cloud-sql-test
cloud-sql-test:
	bash scripts/test-cloud-sql.sh

.PHONY: release-test
release-test:
	node --test scripts/release-manifest.test.mjs scripts/deployment-smoke.test.mjs scripts/migration-receipt.test.mjs
	bash scripts/test-deploy-dev.sh

.PHONY: identity-rehearsal identity-rehearsal-test
identity-rehearsal:
	npm --prefix web run rehearsal:build
	node scripts/identity-rehearsal-server.mjs

identity-rehearsal-test:
	node --test scripts/identity-rehearsal-server.test.mjs
	npm --prefix web run rehearsal:test

.PHONY: observer-test
observer-test:
	bash scripts/test-observer.sh

.PHONY: source-registry-test
source-registry-test:
	bash scripts/test-source-registry.sh

.PHONY: import-service-test
import-service-test:
	bash scripts/test-import-service.sh
