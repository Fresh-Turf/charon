SHELL := /bin/bash
GO := GO15VENDOREXPERIMENT=1 go
OUTPUT_PATH := bin/
INSTALL_PATH := /opt/charon/
SERVICE_PATH := /etc/systemd/system/
SERVICE_NAME := charon.service
NAME := charon
OS := $(shell uname)
MAIN_GO := ./cmd/charon
ROOT_PACKAGE := $(GIT_PROVIDER)/$(ORG)/$(NAME)
GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
PACKAGE_DIRS := $(shell $(GO) list ./... | grep -v /vendor/)
PKGS := $(shell go list ./... | grep -v /vendor | grep -v generated)
PKGS := $(subst  :,_,$(PKGS))
VERSION := $(shell cat VERSION)
BUILDFLAGS := "-s -w -X main.appVersion=$(VERSION)"
CGO_ENABLED = 0
VENDOR_DIR=$(PWD)"/vendor"

all: build

check: fmt build test

build:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -ldflags $(BUILDFLAGS) -o ${OUTPUT_PATH}${NAME} $(MAIN_GO)
	
	cp .env ${OUTPUT_PATH}

test: 
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test $(PACKAGE_DIRS) -test.v

full: $(PKGS)

ifeq ($(OS), Linux)
install: build
install:
	sudo mkdir -p ${INSTALL_PATH}
	sudo cp -r ${OUTPUT_PATH}. ${INSTALL_PATH}
	sudo cp ${SERVICE_NAME} ${SERVICE_PATH}
	sudo systemctl daemon-reload
	sudo systemctl enable ${SERVICE_NAME}

uninstall: 
	systemctl stop ${SERVICE_NAME}
	systemctl disable ${SERVICE_NAME}
	rm ${SERVICE_PATH}/${SERVICE_NAME}
	systemctl daemon-reload
	rm -rf ${INSTALL_PATH}

else
install: 
	@echo "'make install' or 'make uninstall' only work on Linux, sorry"

uninstall: install

endif


fmt:
	@FORMATTED=`$(GO) fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

clean:
	rm -rf build release

linux: export GOOS=linux
linux: build

raspberry: export GOOS=linux
raspberry: export GOARCH=arm
raspberry: OUTPUT_PATH=bin/raspberry/
raspberry: build

.PHONY: release clean

FGT := $(GOPATH)/bin/fgt
$(FGT):
	go get github.com/GeertJohan/fgt

GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	go get github.com/golang/lint/golint

$(PKGS): $(GOLINT) $(FGT)
	@echo "LINTING"
	@$(FGT) $(GOLINT) $(GOPATH)/src/$@/*.go
	@echo "VETTING"
	@go vet -v $@
	@echo "TESTING"
	@go test -v $@

.PHONY: lint
lint: vendor | $(PKGS) $(GOLINT) # ❷
	@cd $(BASE) && ret=0 && for pkg in $(PKGS); do \
	    test -z "$$($(GOLINT) $$pkg | tee /dev/stderr)" || ret=1 ; \
	done ; exit $$ret

watch:
	reflex -r "\.go$" -R "vendor.*" make skaffold-run

skaffold-run: build
	skaffold run -p dev
