include hack/base.mk

.PHONY: build
build:
	go build -trimpath $(GO_TAGS) $(LDFLAGS) -o build/bin/slsactl main.go

.PHONY: test
test:
	go test -race ./...

verify: verify-lint verify-dirty ## Run verification checks.

verify-lint: $(GOLANGCI)
	$(GOLANGCI) run --timeout 2m
