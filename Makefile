VERSION = $(shell git tag -l --contains HEAD | head -n 1)

ifeq ($(VERSION),)
	VERSION = v0.0.0-$(shell git rev-parse --short HEAD)
endif

GO_TAGS = -tags "netgo,osusergo,static_build"
LDFLAGS = -ldflags "-extldflags -static -s -w -X github.com/rancher/slsactl/cmd.version=$(VERSION)"

.PHONY: build
build:
	go build -trimpath $(GO_TAGS) $(LDFLAGS) -o build/bin/slsactl main.go

