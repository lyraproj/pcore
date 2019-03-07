PKGS := $(shell go list ./...)

.PHONY: test
test:
	go test $(PKGS)
