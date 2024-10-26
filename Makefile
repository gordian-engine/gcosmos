# To build the gcosmos binary:
#     make build

# Everything in the above consecutive sequence of comments is printed out as the output of "make help"
# (with leading "# " stripped).
# ("make help" is also the default target, so just running "make" does the same as "make help").

.PHONY: help
help:
	@sed -n '/^#/p; /^[^#]/q; /^$$/q' Makefile | sed 's/^# //'

# Script to update the local copy of the SDK and apply any patches.
.PHONY: deps
deps:
	$(info Makefile: it should be safe to ignore any warnings during this step)
	./_cosmosvendor/sync_sdk.bash

# Just build the gcosmos binary and place it in the current directory.
# (The binary is already in .gitignore.)
.PHONY: build
build: deps
	go build .
	@echo >&2 executable saved to ./gcosmos

.PHONY: install
install: deps
	go install

.PHONY: start
start: build
	./scripts/run_gcosmos.sh
