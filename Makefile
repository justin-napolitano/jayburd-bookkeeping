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
	go build -trimpath -o bin/worker ./cmd/worker
	go build -trimpath -o bin/deliver ./cmd/deliver
	go build -trimpath -o bin/sandbox-qualify ./cmd/sandbox-qualify
	go build -trimpath -o bin/sandbox-probe ./cmd/sandbox-probe
	go build -trimpath -o bin/database-probe ./cmd/database-probe

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
	node --test scripts/sandbox-release.test.mjs scripts/release-manifest.test.mjs scripts/deployment-smoke.test.mjs scripts/migration-receipt.test.mjs scripts/database-probe-receipt.test.mjs
	bash scripts/test-deploy-dev.sh
	node --test scripts/worker-release-manifest.test.mjs scripts/worker-deployment-receipt.test.mjs
	bash scripts/test-deploy-workers.sh

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

.PHONY: sync-state-test
sync-state-test:
	bash scripts/test-sync-state.sh

.PHONY: connection-state-test acquisition-test
connection-state-test:
	bash scripts/test-connection-state.sh

acquisition-test:
	bash scripts/test-acquisition.sh

.PHONY: connection-service-test
connection-service-test:
	bash scripts/test-connection-service.sh

.PHONY: provider-flow-test
provider-flow-test:
	bash scripts/test-provider-flow.sh

.PHONY: connection-browser-test
connection-browser-test:
	bash scripts/test-app-integration.sh --connections

.PHONY: connection-ops-test webhook-test
connection-ops-test:
	bash scripts/test-connection-ops.sh

webhook-test:
	bash scripts/test-webhook.sh

.PHONY: sync-dispatch-test
sync-dispatch-test:
	bash scripts/test-sync-dispatch.sh

.PHONY: worker-bootstrap-test sandbox-qualification-test
worker-bootstrap-test:
	bash scripts/test-worker-bootstrap.sh

sandbox-qualification-test:
	bash scripts/test-sandbox-qualification.sh

.PHONY: sandbox-bootstrap-test
sandbox-bootstrap-test:
	bash scripts/test-sandbox-bootstrap.sh
	bash scripts/test-show-worker-binding.sh
