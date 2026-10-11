.PHONY: check test build backlog packet issue-drafts migration-create migration-seal migration-check migration-test ledger-test web-check web-test web-build web-browser app-integration

check:
	go run ./cmd/project check
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
