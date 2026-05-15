.PHONY: build build-windows build-win7 build-win7-gui build-all clean help

APP_NAME=cors_reverse_proxy
PKG=./cmd/cors_reverse_proxy
CFG_PKG=cors_reverse_proxy/internal/config

# Build-time defaults (override via env / GH secrets)
DEFAULT_TOKEN ?=
DEFAULT_LISTENING ?=
DEFAULT_HTTP_PROXY ?=
DEFAULT_SKIP_TLS ?=

LDFLAGS_BASE=-s -w
LDFLAGS_VARS=$(if $(DEFAULT_TOKEN),-X '$(CFG_PKG).DefaultToken=$(DEFAULT_TOKEN)') \
             $(if $(DEFAULT_LISTENING),-X '$(CFG_PKG).DefaultListening=$(DEFAULT_LISTENING)') \
             $(if $(DEFAULT_HTTP_PROXY),-X '$(CFG_PKG).DefaultHttpProxy=$(DEFAULT_HTTP_PROXY)') \
             $(if $(DEFAULT_SKIP_TLS),-X '$(CFG_PKG).DefaultSkipTLS=$(DEFAULT_SKIP_TLS)')
LDFLAGS=$(LDFLAGS_BASE) $(LDFLAGS_VARS)

build:
	go build -ldflags "$(LDFLAGS)" -o $(APP_NAME) $(PKG)

build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(APP_NAME).exe $(PKG)

# Windows 7 cross-compile via go-legacy-win7. Requires GO_LEGACY_WIN7=/path/to/toolchain.
define WIN7_BUILD
	@if [ -z "$(GO_LEGACY_WIN7)" ]; then \
		echo "Error: GO_LEGACY_WIN7 not set."; \
		echo "Use: ./scripts/build-win7.sh (auto-downloads), or download from"; \
		echo "  https://github.com/thongtech/go-legacy-win7/releases"; \
		exit 1; \
	fi
	GOROOT=$(GO_LEGACY_WIN7) \
	GOCACHE=/tmp/go-legacy-cache \
	GOMODCACHE=/tmp/go-legacy-modcache \
	PATH=$(GO_LEGACY_WIN7)/bin:$(PATH) \
	GOOS=windows GOARCH=$(1) \
	go build -ldflags "$(LDFLAGS) $(2)" -o $(APP_NAME)_win7_$(3).exe $(PKG)
	@echo "Built $(APP_NAME)_win7_$(3).exe"
endef

build-win7-x86:
	$(call WIN7_BUILD,386,,x86)
build-win7-x86-gui:
	$(call WIN7_BUILD,386,-H windowsgui,x86_gui)
build-win7-x64:
	$(call WIN7_BUILD,amd64,,x64)
build-win7-x64-gui:
	$(call WIN7_BUILD,amd64,-H windowsgui,x64_gui)

build-win7: build-win7-x86 build-win7-x64
build-win7-gui: build-win7-x86-gui build-win7-x64-gui

build-all: build-windows build-win7 build-win7-gui
	@echo "All builds completed!"

clean:
	rm -f $(APP_NAME) $(APP_NAME).exe \
		$(APP_NAME)_win7_x86.exe $(APP_NAME)_win7_x64.exe \
		$(APP_NAME)_win7_x86_gui.exe $(APP_NAME)_win7_x64_gui.exe

help:
	@echo "Targets:"
	@echo "  build              - Native build"
	@echo "  build-windows      - Windows x64 (Windows 10+)"
	@echo "  build-win7-x86 / -x64 / -x86-gui / -x64-gui  - Win7 (needs GO_LEGACY_WIN7)"
	@echo "  build-win7 / -gui  - All Win7 console / GUI"
	@echo "  build-all          - All Windows variants"
	@echo "  clean              - Remove build artifacts"
	@echo ""
	@echo "Easy: ./scripts/build-win7.sh  (GUI=1, GOARCH=amd64 supported)"
