PNPM ?= pnpm

.PHONY: build dev gen lint seed test typecheck go-build go-vet go-test check

dev:
	$(PNPM) dev

build:
	$(PNPM) build

lint:
	$(PNPM) lint

typecheck:
	$(PNPM) typecheck

test:
	$(PNPM) test

gen:
	$(PNPM) gen

seed:
	$(PNPM) seed

go-build:
	go build ./...

go-vet:
	go vet ./...

go-test:
	go test ./...

check: lint typecheck test build go-build go-vet go-test
