# Fyne uses CGO (GLFW). Cross-compiling to another OS requires a matching C cross-compiler
# (e.g. zig cc, musl-cross, mingw-w64) and often CC/CXX set per target; native runs on the host OS.
# CI matrix builds (one runner per OS) are the usual approach.

DIST := dist
APP := v2ray-subscription-monitor
APP_CLI := v2ray-subscription-cli
CLI_PKG := ./cmd/v2ray-subscription-cli

# Smaller binaries: strip symbol/DWARF tables, trim module paths, omit VCS metadata.
# CGO_CFLAGS=-Os asks the C compiler to favor size for GLFW/native glue (clang/gcc).
GO_RELEASE := -buildvcs=false -trimpath
GO_LDFLAGS_STRIP := -ldflags="-s -w"
# Suppress Apple ld warning: duplicate -lobjc (darwin only).
GO_LDFLAGS_DARWIN := -ldflags="-s -w -extldflags=-Wl,-no_warn_duplicate_libraries"

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  GO_BUILD_LOCAL := $(GO_RELEASE) $(GO_LDFLAGS_DARWIN)
else
  GO_BUILD_LOCAL := $(GO_RELEASE) $(GO_LDFLAGS_STRIP)
endif

# Extra env for GUI dist/build (Fyne).
CGO_RELEASE := CGO_ENABLED=1 CGO_CFLAGS=-Os

.PHONY: build build-debug build-cli install dist dist-all dist-cli-all clean-dist \
	dist-darwin dist-linux dist-windows \
	dist-darwin-amd64 dist-darwin-arm64 \
	dist-linux-amd64 dist-linux-arm64 \
	dist-windows-amd64 dist-windows-arm64 \
	dist-cli-darwin dist-cli-linux dist-cli-windows \
	dist-cli-darwin-amd64 dist-cli-darwin-arm64 \
	dist-cli-linux-amd64 dist-cli-linux-arm64 \
	dist-cli-windows-amd64 dist-cli-windows-arm64

# Default: stripped GUI binary (-s -w, -trimpath, -buildvcs=false; CGO -Os).
build:
	$(CGO_RELEASE) go build $(GO_BUILD_LOCAL) -o $(APP) .

# Pure Go CLI (~5–6 MiB stripped): same fetch logic, no window/tray.
build-cli:
	CGO_ENABLED=0 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $(APP_CLI) $(CLI_PKG)

# Unstripped GUI binary for profiling / gdb / crash symbols.
build-debug:
	go build -o $(APP) .

install:
	$(CGO_RELEASE) go install $(GO_BUILD_LOCAL) .

dist: dist-all

dist-all: \
	$(DIST)/$(APP)-darwin-amd64 \
	$(DIST)/$(APP)-darwin-arm64 \
	$(DIST)/$(APP)-linux-amd64 \
	$(DIST)/$(APP)-linux-arm64 \
	$(DIST)/$(APP)-windows-amd64.exe \
	$(DIST)/$(APP)-windows-arm64.exe

# Headless probe; CGO_ENABLED=0 — easy cross-compile from any OS.
dist-cli-all: \
	$(DIST)/$(APP_CLI)-darwin-amd64 \
	$(DIST)/$(APP_CLI)-darwin-arm64 \
	$(DIST)/$(APP_CLI)-linux-amd64 \
	$(DIST)/$(APP_CLI)-linux-arm64 \
	$(DIST)/$(APP_CLI)-windows-amd64.exe \
	$(DIST)/$(APP_CLI)-windows-arm64.exe

dist-darwin: dist-darwin-amd64 dist-darwin-arm64
dist-linux: dist-linux-amd64 dist-linux-arm64
dist-windows: dist-windows-amd64 dist-windows-arm64

dist-darwin-amd64: $(DIST)/$(APP)-darwin-amd64
dist-darwin-arm64: $(DIST)/$(APP)-darwin-arm64
dist-linux-amd64: $(DIST)/$(APP)-linux-amd64
dist-linux-arm64: $(DIST)/$(APP)-linux-arm64
dist-windows-amd64: $(DIST)/$(APP)-windows-amd64.exe
dist-windows-arm64: $(DIST)/$(APP)-windows-arm64.exe

dist-cli-darwin: dist-cli-darwin-amd64 dist-cli-darwin-arm64
dist-cli-linux: dist-cli-linux-amd64 dist-cli-linux-arm64
dist-cli-windows: dist-cli-windows-amd64 dist-cli-windows-arm64

dist-cli-darwin-amd64: $(DIST)/$(APP_CLI)-darwin-amd64
dist-cli-darwin-arm64: $(DIST)/$(APP_CLI)-darwin-arm64
dist-cli-linux-amd64: $(DIST)/$(APP_CLI)-linux-amd64
dist-cli-linux-arm64: $(DIST)/$(APP_CLI)-linux-arm64
dist-cli-windows-amd64: $(DIST)/$(APP_CLI)-windows-amd64.exe
dist-cli-windows-arm64: $(DIST)/$(APP_CLI)-windows-arm64.exe

$(DIST)/$(APP)-darwin-amd64:
	mkdir -p $(DIST)
	$(CGO_RELEASE) GOOS=darwin GOARCH=amd64 go build $(GO_RELEASE) $(GO_LDFLAGS_DARWIN) -o $@ .

$(DIST)/$(APP)-darwin-arm64:
	mkdir -p $(DIST)
	$(CGO_RELEASE) GOOS=darwin GOARCH=arm64 go build $(GO_RELEASE) $(GO_LDFLAGS_DARWIN) -o $@ .

$(DIST)/$(APP)-linux-amd64:
	mkdir -p $(DIST)
	$(CGO_RELEASE) GOOS=linux GOARCH=amd64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ .

$(DIST)/$(APP)-linux-arm64:
	mkdir -p $(DIST)
	$(CGO_RELEASE) GOOS=linux GOARCH=arm64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ .

$(DIST)/$(APP)-windows-amd64.exe:
	mkdir -p $(DIST)
	$(CGO_RELEASE) GOOS=windows GOARCH=amd64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ .

$(DIST)/$(APP)-windows-arm64.exe:
	mkdir -p $(DIST)
	$(CGO_RELEASE) GOOS=windows GOARCH=arm64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ .

$(DIST)/$(APP_CLI)-darwin-amd64:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ $(CLI_PKG)

$(DIST)/$(APP_CLI)-darwin-arm64:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ $(CLI_PKG)

$(DIST)/$(APP_CLI)-linux-amd64:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ $(CLI_PKG)

$(DIST)/$(APP_CLI)-linux-arm64:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ $(CLI_PKG)

$(DIST)/$(APP_CLI)-windows-amd64.exe:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ $(CLI_PKG)

$(DIST)/$(APP_CLI)-windows-arm64.exe:
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ $(CLI_PKG)

clean-dist:
	rm -rf $(DIST)
