VERSION = $(shell git tag -l --contains HEAD | head -n 1)

ifeq ($(VERSION),)
	VERSION = v0.0.0-$(shell git rev-parse --short HEAD)
endif

GO_TAGS = -tags "netgo,osusergo"
LDFLAGS = -ldflags "-extldflags -s -w -X github.com/rancherlabs/slsactl/cmd.version=$(VERSION)"

.PHONY: build
build:
	go build -trimpath $(GO_TAGS) $(LDFLAGS) -o build/bin/slsactl main.go

