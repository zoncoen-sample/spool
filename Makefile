export GO111MODULE=on

GOPATH := $(shell go env GOPATH)
export GOBIN=$(shell pwd -P)/bin
export PATH=$(GOBIN):$(shell echo $$PATH)

export SPANNER_PROJECT_ID=gcpug-public-spanner
export SPANNER_INSTANCE_ID=merpay-sponsored-instance
export SPOOL_SPANNER_DATABASE_ID=zoncoen-spool
export SPOOL_DATABASE_PREFIX=zoncoen-spool
export SCHEMA=./db/schema.sql

DATABASE_ID :=

.PHONY: test
test:
	USE_SPANNER_EMULATOR=true go test . -v

.PHONY: test/ci
test/ci:
	SPANNER_DATABASE_ID=$(DATABASE_ID) go test . -v

.PHONY: tools
tools:
	go install \
		go.mercari.io/yo \
		github.com/gcpug/spool/cmd/spool

.PHONY: gen
gen: db/schema.sql
	$(GOBIN)/yo generate db/schema.sql --from-ddl -o models

.PHONY: spool/setup
spool/setup:
	$(GOBIN)/spool setup --schema ${SCHEMA}

.PHONY: spool/get-or-create
spool/get-or-create:
	@$(GOBIN)/spool get-or-create --schema ${SCHEMA} --db-name-prefix=${SPOOL_DATABASE_PREFIX}

.PHONY: spool/put
spool/put:
	@$(GOBIN)/spool put --schema ${SCHEMA} ${DATABASE_ID}

.PHONY: spool/list
spool/list:
	@$(GOBIN)/spool list --all

.PHONY: spool/clean
spool/clean:
	@$(GOBIN)/spool clean --all --force --ignore-used-within-days=7
	@echo alive databases
	@make spool/list
