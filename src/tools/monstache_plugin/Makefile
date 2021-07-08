TARGET  = monstache-plugin

# build
PWD         = $(shell pwd)
LOCALBUILD  = $(PWD)/build
OUTPUT_DIR ?= $(LOCALBUILD)

# monstache
MONSTACHE = $(PWD)/vendor/github.com/rwynn/monstache

# package
BINDIR = ${OUTPUT_DIR}/monstache_plugin

default:
	go build -buildmode=plugin -o $(BINDIR)/etc/$(TARGET).so *.go
	go build -ldflags="-s -w" -v -o $(BINDIR)/monstache $(MONSTACHE)/monstache.go
	@cp -rf etc/config.toml $(BINDIR)/etc
	@cp -rf monstache.sh $(BINDIR)
	@cp -rf CHANGELOG.md $(BINDIR)
	@cp -rf README.md $(BINDIR)

clean:
	@rm -rf ${BINDIR} $(LOCALBUILD)
