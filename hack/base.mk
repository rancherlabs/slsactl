GOLANGCI_VERSION ?= 8f3b0c7ed018e57905fbd873c697e0b1ede605a5 # v2.11.4
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
	$(call go-install-tool,$(GOLANGCI),github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION))

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
