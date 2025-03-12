GOLANGCI_VERSION ?= v1.64.7
TOOLS_BIN := $(shell mkdir -p build/tools && realpath build/tools)

VERSION = $(shell git tag -l --contains HEAD | head -n 1)
CHANGES = $(shell git status --porcelain --untracked-files=no)
ifneq ($(CHANGES),)
	DIRTY = -dirty
endif

ifeq ($(VERSION),)
	VERSION = v0.0.0-$(shell git rev-parse --short HEAD)$(DIRTY)
endif

GO_TAGS = -tags "netgo,osusergo"
LDFLAGS = -ldflags "-extldflags -s -w -X github.com/rancherlabs/slsactl/cmd.version=$(VERSION)"

GOLANGCI = $(TOOLS_BIN)/golangci-lint-$(GOLANGCI_VERSION)
$(GOLANGCI):
	rm -f $(TOOLS_BIN)/golangci-lint*
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_VERSION)/install.sh | sh -s -- -b $(TOOLS_BIN) $(GOLANGCI_VERSION)
	mv $(TOOLS_BIN)/golangci-lint $(TOOLS_BIN)/golangci-lint-$(GOLANGCI_VERSION)

# go-install-tool will 'go install' any package $2 and install it as $1.
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOBIN=$(TOOLS_BIN) go install $(2) ;\
}
endef

verify-dirty:
ifneq ($(shell git status --porcelain --untracked-files=no),)
	@echo worktree is dirty
	@git --no-pager status
	@git --no-pager diff
	@exit 1
endif
