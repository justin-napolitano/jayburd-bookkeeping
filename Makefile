.PHONY: check test build backlog packet issue-drafts

check:
	go run ./cmd/project check
	bash scripts/check-stage-runbook.sh --validate
	go vet ./...
	git diff --check

test:
	go test -race ./...

build:
	go build -trimpath -o bin/project ./cmd/project

backlog:
	go run ./cmd/project backlog

packet:
	go run ./cmd/project packet $(ID)

issue-drafts:
	go run ./cmd/project drafts
