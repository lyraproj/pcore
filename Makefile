PKGS := $(shell go list ./...)

.PHONY: test
test:
	go test $(PKGS)

.PHONY: lint
lint:
	golangci-lint run
