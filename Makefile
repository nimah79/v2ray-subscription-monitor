# Fyne uses CGO (GLFW). Cross-compiling to another OS requires a matching C cross-compiler
# (e.g. zig cc, musl-cross, mingw-w64) and often CC/CXX set per target; native runs on the host OS.
# CI matrix builds (one runner per OS) are the usual approach.

DIST := dist
APP := v2ray-subscription-monitor
APP_CLI := v2ray-subscription-cli
CLI_PKG := ./cmd/v2ray-subscription-cli

# macOS .app / .dmg: fyne package, then sindresorhus/create-dmg (Applications alias + drag-and-drop layout).
# Requires Darwin + Node.js (for npx). https://github.com/sindresorhus/create-dmg
MACOS_APP_TITLE := V2Ray Subscription Monitor
MACOS_APP_ID := io.github.v2ray-subscription-data-usage-monitor
MACOS_ICON := $(CURDIR)/assets/icons/v2ray-subscription-monitor.png
FYNE ?= go run fyne.io/fyne/v2/cmd/fyne@v2.7.3
CREATE_DMG ?= npx --yes create-dmg@8
# Nearest ancestor tag (same as release ref when building from a tag). Override in CI: APP_VERSION=… APP_BUILD=…
GIT_NEAREST_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)
APP_VERSION ?= $(if $(strip $(GIT_NEAREST_TAG)),$(patsubst v%,%,$(GIT_NEAREST_TAG)),dev)
APP_BUILD ?= 1

# Injected into the GUI binary (main.appVersion); keep in sync with APP_VERSION / release workflow.
GO_X_APP_VERSION := -X 'main.appVersion=$(APP_VERSION)'

# Smaller binaries: strip symbol/DWARF tables, trim module paths, omit VCS metadata.
# CGO_CFLAGS=-Os asks the C compiler to favor size for GLFW/native glue (clang/gcc).
GO_RELEASE := -buildvcs=false -trimpath
GO_LDFLAGS_STRIP := -ldflags="-s -w $(GO_X_APP_VERSION)"
# Suppress Apple ld warning: duplicate -lobjc (darwin only).
GO_LDFLAGS_DARWIN := -ldflags="-s -w $(GO_X_APP_VERSION) -extldflags=-Wl,-no_warn_duplicate_libraries"

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
	dist-darwin-amd64.dmg dist-darwin-arm64.dmg \
	dist-linux-amd64 dist-linux-arm64 \
	dist-windows-amd64 dist-windows-arm64 \
	dist-windows-amd64-setup dist-windows-arm64-setup dist-windows-installers \
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
	go build -ldflags="$(GO_X_APP_VERSION)" -o $(APP) .

install:
	$(CGO_RELEASE) go install $(GO_BUILD_LOCAL) .

dist: dist-all

ifeq ($(UNAME_S),Darwin)
dist-all: \
	$(DIST)/$(APP)-darwin-amd64.dmg \
	$(DIST)/$(APP)-darwin-arm64.dmg \
	$(DIST)/$(APP)-linux-amd64.AppImage \
	$(DIST)/$(APP)-linux-arm64.AppImage \
	$(DIST)/$(APP)-windows-amd64.exe \
	$(DIST)/$(APP)-windows-arm64.exe
else
dist-all: \
	$(DIST)/$(APP)-darwin-amd64 \
	$(DIST)/$(APP)-darwin-arm64 \
	$(DIST)/$(APP)-linux-amd64.AppImage \
	$(DIST)/$(APP)-linux-arm64.AppImage \
	$(DIST)/$(APP)-windows-amd64.exe \
	$(DIST)/$(APP)-windows-arm64.exe
endif

# Windows GUI installers (Inno Setup 6). Prereqs merge with dist-all when OS=Windows_NT.
ifeq ($(OS),Windows_NT)
dist-all: $(DIST)/$(APP)-windows-amd64-setup.exe $(DIST)/$(APP)-windows-arm64-setup.exe
endif

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

ifeq ($(UNAME_S),Darwin)
dist-darwin-amd64: $(DIST)/$(APP)-darwin-amd64.dmg
dist-darwin-arm64: $(DIST)/$(APP)-darwin-arm64.dmg
dist-darwin-amd64.dmg: $(DIST)/$(APP)-darwin-amd64.dmg
dist-darwin-arm64.dmg: $(DIST)/$(APP)-darwin-arm64.dmg
else
dist-darwin-amd64: $(DIST)/$(APP)-darwin-amd64
dist-darwin-arm64: $(DIST)/$(APP)-darwin-arm64
dist-darwin-amd64.dmg dist-darwin-arm64.dmg:
	@echo >&2 "DMG packaging requires macOS + Node.js (fyne package + npx create-dmg)." && exit 1
endif
dist-linux-amd64: $(DIST)/$(APP)-linux-amd64.AppImage
dist-linux-arm64: $(DIST)/$(APP)-linux-arm64.AppImage
dist-windows-amd64: $(DIST)/$(APP)-windows-amd64.exe
dist-windows-arm64: $(DIST)/$(APP)-windows-arm64.exe

ifeq ($(OS),Windows_NT)
dist-windows: $(DIST)/$(APP)-windows-amd64-setup.exe $(DIST)/$(APP)-windows-arm64-setup.exe

dist-windows-amd64-setup: $(DIST)/$(APP)-windows-amd64-setup.exe
dist-windows-arm64-setup: $(DIST)/$(APP)-windows-arm64-setup.exe
dist-windows-installers: dist-windows-amd64-setup dist-windows-arm64-setup
else
dist-windows-amd64-setup dist-windows-arm64-setup dist-windows-installers:
	@echo >&2 "Windows setup targets require Windows with Inno Setup 6 (ISCC). See https://jrsoftware.org/isinfo.php" && exit 1
endif

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

ifeq ($(UNAME_S),Darwin)
# create-dmg: window with app + /Applications shortcut (https://github.com/sindresorhus/create-dmg).
# --no-code-sign: CI has no Apple cert; users can still open the DMG. --no-version-in-filename: predictable name for mv.
$(DIST)/$(APP)-darwin-%.dmg: $(DIST)/$(APP)-darwin-%
	rm -rf $(DIST)/dmg-tmp-$*
	mkdir -p $(DIST)/dmg-tmp-$*
	cp $(DIST)/$(APP)-darwin-$* $(DIST)/dmg-tmp-$*/$(APP)
	cd $(DIST)/dmg-tmp-$* && $(FYNE) package -os darwin \
		-executable ./$(APP) \
		-name "$(MACOS_APP_TITLE)" \
		-appID $(MACOS_APP_ID) \
		-icon $(MACOS_ICON) \
		-appVersion $(APP_VERSION) \
		-appBuild $(APP_BUILD)
	cd $(DIST)/dmg-tmp-$* && $(CREATE_DMG) --overwrite --no-code-sign --no-version-in-filename "$(MACOS_APP_TITLE).app" .
	mv "$(DIST)/dmg-tmp-$*/$(MACOS_APP_TITLE).dmg" "$@"
	rm -rf $(DIST)/dmg-tmp-$*
	rm -f $(DIST)/$(APP)-darwin-$*
endif

# Linux GUI release artifact is an AppImage (linuxdeploy + appimagetool). Host arch must match target.
ifeq ($(UNAME_S),Linux)
$(DIST)/$(APP)-linux-%.AppImage:
	bash "$(CURDIR)/installer/linux/build-appimage.sh" "$*" "$(CURDIR)" "$(CURDIR)/$(DIST)" "$(APP_VERSION)"
else
$(DIST)/$(APP)-linux-amd64.AppImage $(DIST)/$(APP)-linux-arm64.AppImage:
	@echo >&2 "Linux GUI releases are AppImages; build on Linux with matching arch (see installer/linux/build-appimage.sh)." && exit 1
endif

$(DIST)/$(APP)-windows-amd64.exe:
	mkdir -p $(DIST)
	$(CGO_RELEASE) GOOS=windows GOARCH=amd64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ .

$(DIST)/$(APP)-windows-arm64.exe:
	mkdir -p $(DIST)
	$(CGO_RELEASE) GOOS=windows GOARCH=arm64 go build $(GO_RELEASE) $(GO_LDFLAGS_STRIP) -o $@ .

# Inno Setup 6 (https://jrsoftware.org/isinfo.php). Install on Windows or set ISCC to ISCC.exe.
ifeq ($(OS),Windows_NT)
ISCC ?= C:/Program Files (x86)/Inno Setup 6/ISCC.exe
SETUP_ISS := $(CURDIR)/installer/windows/setup.iss

$(DIST)/$(APP)-windows-amd64-setup.exe: $(DIST)/$(APP)-windows-amd64.exe
	"$(ISCC)" /DBuildArch=amd64 "/DMyAppVersion=$(APP_VERSION)" "$(SETUP_ISS)"

$(DIST)/$(APP)-windows-arm64-setup.exe: $(DIST)/$(APP)-windows-arm64.exe
	"$(ISCC)" /DArm64=1 /DBuildArch=arm64 "/DMyAppVersion=$(APP_VERSION)" "$(SETUP_ISS)"

else

$(DIST)/$(APP)-windows-amd64-setup.exe $(DIST)/$(APP)-windows-arm64-setup.exe:
	@echo >&2 "Windows setup.exe requires Windows with Inno Setup 6 (ISCC). See https://jrsoftware.org/isinfo.php — on Windows set ISCC if installed elsewhere." && exit 1

endif

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
