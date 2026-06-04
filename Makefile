BIN := bin/neotrinkey
PKG := ./cmd/module

# All target platforms.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

$(BIN): Makefile go.mod $(wildcard trinkey/*.go) cmd/module/*.go
	GOOS=$(VIAM_BUILD_OS) GOARCH=$(VIAM_BUILD_ARCH) go build -o $(BIN) $(PKG)

lint:
	gofmt -s -w .

test:
	go test ./...

update:
	go get go.viam.com/rdk@latest
	go mod tidy

# Single-platform tarball for the current build (used by cloud build / upload).
module.tar.gz: meta.json $(BIN)
	tar czf $@ meta.json README.md firmware $(BIN)

# Cross-compile every platform and produce one tarball per platform:
# dist/neotrinkey-<os>-<arch>.tar.gz
build-all: test
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		echo "building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build -o bin/neotrinkey$$ext $(PKG) || exit 1; \
		tar czf dist/neotrinkey-$$os-$$arch.tar.gz meta.json README.md firmware bin/neotrinkey$$ext; \
		rm -f bin/neotrinkey$$ext; \
	done
	@echo "--- artifacts ---"; ls -1 dist

clean:
	rm -rf bin dist module.tar.gz

all: test module.tar.gz
