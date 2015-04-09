SHELL := /bin/bash
PKG := github.com/Clever/kayvee-go
SUBPKG_NAMES :=
SUBPKGS = $(addprefix $(PKG)/, $(SUBPKG_NAMES))
PKGS = $(PKG) $(SUBPKGS)
GODEP := $(GOPATH)/bin/godep

.PHONY: test golint README

test: docs tests.json $(PKGS)

golint:
	@go get github.com/golang/lint/golint

$(GODEP):
	go get github.com/tools/godep

README.md: *.go
	@go get github.com/robertkrimen/godocdown/godocdown
	@godocdown $(PKG) > README.md

$(PKGS): golint docs $(GODEP)
	@gofmt -w=true $(GOPATH)/src/$@*/**.go
ifneq ($(NOLINT),1)
	@echo "LINTING..."
	@PATH=$(PATH):$(GOPATH)/bin golint $(GOPATH)/src/$@*/**.go
	@echo ""
endif
ifeq ($(COVERAGE),1)
	$(GODEP) go test -cover -coverprofile=$(GOPATH)/src/$@/c.out $@ -test.v
	$(GODEP) go tool cover -html=$(GOPATH)/src/$@/c.out
else
	@echo "TESTING..."
	$(GODEP) go test $@ -test.v
	@echo ""
endif

docs: $(addsuffix /README.md, $(SUBPKG_NAMES)) README.md
%/README.md: PATH := $(PATH):$(GOPATH)/bin
%/README.md: %/*.go
	@go get github.com/robertkrimen/godocdown/godocdown
	@godocdown $(PKG)/$(shell dirname $@) > $@

tests.json:
	wget https://raw.githubusercontent.com/Clever/kayvee/master/tests.json -O test/tests.json
