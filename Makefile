SHELL := /bin/sh

LOCAL_BIN := $(CURDIR)/.local/bin
GO_CACHE := $(CURDIR)/.local/cache/go-build
SWAG := $(LOCAL_BIN)/swag
SWAG_VERSION := v1.16.4
GVA_GO_PROXY ?= https://goproxy.cn,direct

.PHONY: init tools up down restart ps logs credentials swagger swagger-check backend-check backend-test frontend-check frontend-lint frontend-build phase2-rehearsal-check verify reset

init:
	@./scripts/init-local-env.sh

tools:
	@mkdir -p "$(LOCAL_BIN)" "$(GO_CACHE)"
	@if [ ! -x "$(SWAG)" ] || ! "$(SWAG)" --version | grep -q '$(SWAG_VERSION)'; then \
		cd server && GOBIN="$(LOCAL_BIN)" GOCACHE="$(GO_CACHE)" GOPROXY="$(GVA_GO_PROXY)" go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION); \
	fi

up: init
	docker compose up -d --build --wait
	@./scripts/init-database.sh
	@docker compose restart server
	@docker compose up -d --wait

down:
	docker compose down

restart:
	docker compose restart

ps:
	docker compose ps

logs:
	docker compose logs -f --tail=200

credentials:
	@./scripts/show-local-credentials.sh

swagger: tools
	@mkdir -p "$(GO_CACHE)"
	@cd server && GOCACHE="$(GO_CACHE)" "$(SWAG)" init -g main.go -o docs

swagger-check: tools
	@mkdir -p "$(GO_CACHE)" "$(CURDIR)/.local/swagger-check/docs"
	@cd server && GOCACHE="$(GO_CACHE)" "$(SWAG)" init -g main.go -o "$(CURDIR)/.local/swagger-check/docs" >/dev/null
	@diff -q server/docs/docs.go .local/swagger-check/docs/docs.go
	@diff -q server/docs/swagger.json .local/swagger-check/docs/swagger.json
	@diff -q server/docs/swagger.yaml .local/swagger-check/docs/swagger.yaml

backend-check:
	@mkdir -p "$(GO_CACHE)"
	@cd server && GOCACHE="$(GO_CACHE)" go test -mod=readonly ./config ./core -count=1
	@cd server && GOCACHE="$(GO_CACHE)" go test -mod=readonly ./... -run '^$$' -count=1 -vet=off

backend-test:
	@mkdir -p "$(GO_CACHE)"
	@cd server && GOCACHE="$(GO_CACHE)" go test -mod=readonly ./... -count=1

frontend-check:
	docker compose --env-file .env.example build web

frontend-lint: frontend-check

frontend-build: frontend-check

phase2-rehearsal-check:
	@mkdir -p "$(GO_CACHE)"
	@cd server && GOCACHE="$(GO_CACHE)" go test -mod=readonly ./config -run '^TestPhaseTwoRehearsalContract$$' -count=1

verify: backend-check swagger-check frontend-check
	docker compose --env-file .env.example config --quiet

reset:
	@printf '%s\n' 'This removes all local MySQL, Redis, server config, upload, and log volumes.'
	@printf '%s' 'Type RESET to continue: '
	@read answer; test "$$answer" = RESET
	docker compose down --volumes --remove-orphans
