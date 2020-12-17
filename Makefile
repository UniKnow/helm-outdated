PLUGIN_NAME := outdated

VERSION = $(shell cat <plugin.yaml |  grep version: | cut -d: -f2 | tr -d ' "')
$(info Executing make for Helm outdated ${VERSION})

# Temporary directory for tools
TOOLS_BIN_DIR = $(shell pwd)/tmp/bin

.PHONY: build
build: build_linux build_mac build_windows
	@echo Finished building ${VERSION} of Helm outdated

build_windows: export GOARCH=amd64
build_windows:
	@GOOS=windows go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/windows/amd64/helm-outdated main.go  # windows

link_windows:
	@cp bin/windows/amd64/helm-outdated ./bin/helm-outdated

build_linux: export GOARCH=amd64
build_linux: export CGO_ENABLED=0
build_linux:
	@GOOS=linux go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
		-o bin/linux/amd64/helm-outdated main.go  # linux

link_linux:
	@cp bin/linux/amd64/helm-outdated ./bin/helm-outdated

build_mac: export GOARCH=amd64
build_mac: export CGO_ENABLED=0
build_mac:
	@GOOS=darwin go build -v --ldflags="-w -X main.Version=$(VERSION) -X main.Revision=$(REVISION)" \
                 		-o bin/darwin/amd64/helm-outdated main.go # mac osx
	@cp bin/darwin/amd64/helm-outdated ./bin/helm-outdated # For use w make install

link_mac:
	@cp bin/darwin/amd64/helm-outdated ./bin/helm-outdated

.PHONY: clean
clean:
	@git status --ignored --short | grep '^!! ' | sed 's/!! //' | xargs rm -rf
	rm -rf $(TOOLS_BIN_DIR)

.PHONY: tree
tree:
	@tree -I vendor

git-tag-release: check-release-version
	@echo Creating tag for ${VERSION}
	git config --global user.name "UniKnow"
	git tag --annotate ${VERSION} --message "helm-outdated ${VERSION}"

check-release-version:
	if test x$$(git tag --list ${VERSION}) != x; \
	then \
		echo "Tag [${VERSION}] already exists. Please check the working copy."; git diff . ; exit 1;\
	fi

$(TOOLS_BIN_DIR):
	mkdir -p $(TOOLS_BIN_DIR)

.PHONY: release
release: git-tag-release goreleaser

	@scripts/release.sh $(VERSION)

.PHONY: install
install:
	HELM_OUTDATED_DEPENDENCIES_PLUGIN_NO_INSTALL_HOOK=1 helm plugin install $(shell pwd)

.PHONY: remove
remove:
	helm plugin remove $(PLUGIN_NAME)

goreleaser: GORELEASER_VERSION=v0.137.0
goreleaser: $(TOOLS_BIN_DIR)
ifeq (,$(wildcard $(TOOLS_BIN_DIR)/goreleaser))
	@scripts/goreleaser.sh -b $(TOOLS_BIN_DIR) ${GORELEASER_VERSION}
endif
GORELEASER=$(TOOLS_BIN_DIR)/goreleaser