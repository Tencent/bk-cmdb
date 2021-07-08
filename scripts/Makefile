SHELL=bash
PWD=$(shell pwd)

space:=$()
comma:=,

ARCH?=$(shell uname -m)

# Version  : v3.2.2
# Tag          : release-v3.2.2
# BuildTime: 2019-02-19T16:48:43+0800
# GitHash  : 3185b32956a6e59df1ee0744c6edf72e09828778
# RunMode  : product

BUILDTIME?=$(shell date +%Y-%m-%dT%T%z)
GITHASH?=$(shell git rev-parse HEAD)
GITTAG?=$(shell git describe --all --tags --exact-match 2>/dev/null | awk -F '/' '{print $$NF}')
VERSION?=$(shell git symbolic-ref -q --short HEAD || git describe --all --tags --exact-match 2>/dev/null | awk -F '-' '{print $$NF}')
ISDEBUG?=false
CCDistro?=community
ifeq ($(ISDEBUG),true)
	GOBUILD_FLAGS = -ldflags "-X configcenter/src/common/version.CCRunMode=dev -X\
	configcenter/src/common/version.CCVersion=${VERSION} -X configcenter/src/common/version.CCBuildTime=${BUILDTIME} -X\
	configcenter/src/common/version.CCGitHash=${GITHASH} -X configcenter/src/common/version.CCTag=${GITTAG} -X\
	configcenter/src/common/version.CCDistro=${CCDistro}"
else
	GOBUILD_FLAGS = -ldflags "-X configcenter/src/common/version.CCVersion=${VERSION} -X\
	configcenter/src/common/version.CCBuildTime=${BUILDTIME} -X configcenter/src/common/version.CCGitHash=${GITHASH} -X\
	configcenter/src/common/version.CCTag=${GITTAG} -X configcenter/src/common/version.CCDistro=${CCDistro}"
endif

SCRIPT_DIR?=$(realpath $(firstword $(subst :, ,${GOPATH}))/src/configcenter/scripts)
RESOURCE_DIR?=$(realpath $(firstword $(subst :, ,${GOPATH}))/src/configcenter/resources)
SOURCE_ROOT?=$(realpath $(firstword $(subst :, ,${GOPATH}))/src/configcenter/src)
BUILD_DIR?=$(SOURCE_ROOT)/bin/build/$(VERSION)
BIN_PATH?=$(BUILD_DIR)
export OUTPUT_DIR = ${BIN_PATH}

GOFILES = $(shell find . -type f -name '*.go')
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "*/vendor/*")
GOFILES_ALL = $(GOFILES)
NOW=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# available
TARGET_NAME?=
TARGET_PORT?=9999

PREPARE_CFG?=true

# building
${TARGET_NAME}: $(GOFILES_ALL)
ifeq ($(PREPARE_CFG),true)
	@python $(SCRIPT_DIR)/prepare_cfg.py $(BIN_PATH) $(TARGET_NAME) $(RESOURCE_DIR) $(SCRIPT_DIR) $(TARGET_PORT) $(VERSION)
	@echo
endif
	@echo -e "\033[34mbuilding the target: $@... \033[0m"
	@echo 
	go build -i $(GOBUILD_FLAGS) -o $(BIN_PATH)/$(TARGET_NAME)/$(TARGET_NAME)
	@chmod +x  $(BIN_PATH)/$(TARGET_NAME)/$(TARGET_NAME)
