include hack/base.mk

.PHONY: build
build:
	go build -trimpath $(GO_TAGS) $(LDFLAGS) -o build/bin/slsactl main.go

.PHONY: test
test:
	go test -race ./...

e2e:
	./hack/e2e.sh

verify: verify-lint verify-dirty ## Run verification checks.

verify-lint: $(GOLANGCI)
	$(GOLANGCI) run --timeout 10m
