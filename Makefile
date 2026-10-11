BEAR_CLI_OUTPUT ?= bear
BEAR_CLI_TARGET_DIR ?= /usr/local/bin
BEAR_CLI_DEPLOY_DIR ?= /usr/local/bin
BEAR_CLI_WORK_DIR ?= ./
BEAR_MENU_BAR_NAME ?= Bear Menu Bar
BEAR_MENU_BAR_BINARY ?= bear-menu-bar
BEAR_MENU_BAR_BUILD_DIR ?= dist
BEAR_MENU_BAR_APP ?= $(BEAR_MENU_BAR_BUILD_DIR)/$(BEAR_MENU_BAR_NAME).app
BEAR_MENU_BAR_ICON_SOURCE ?= cmd/bear-menu-bar/assets/bear-app-icon.png
BEAR_MENU_BAR_ICONSET ?= $(BEAR_MENU_BAR_BUILD_DIR)/BearIcon.iconset
BEAR_MENU_BAR_INSTALL_DIR ?= $(HOME)/Applications
BEAR_MENU_BAR_DEPLOY_DIR ?= /Applications

.PHONY: build build-all build-cli build-menu-bar install install-menu-bar deploy deploy-menu-bar run-menu-bar clean clean-menu-bar reinstall reinstall-menu-bar

build: build-cli

build-all: build-cli build-menu-bar

build-cli:
	cd $(BEAR_CLI_WORK_DIR) && go build -trimpath -o $(BEAR_CLI_OUTPUT) ./cmd/bear

install: build-cli
	cp $(BEAR_CLI_OUTPUT) $(BEAR_CLI_TARGET_DIR)
	@echo "Installed $(BEAR_CLI_OUTPUT) to $(BEAR_CLI_TARGET_DIR)"

deploy: build-cli
	@if [ "$$(uname -s)" != "Darwin" ]; then \
		echo "Error: deploy is only supported on macOS"; \
		exit 1; \
	fi
	sudo install -d $(BEAR_CLI_DEPLOY_DIR)
	sudo install -m 0755 $(BEAR_CLI_OUTPUT) $(BEAR_CLI_DEPLOY_DIR)/$(BEAR_CLI_OUTPUT)
	@echo "Deployed $(BEAR_CLI_OUTPUT) as $(BEAR_CLI_DEPLOY_DIR)/$(BEAR_CLI_OUTPUT)"
	@echo "Run '$(BEAR_CLI_OUTPUT) --help' to verify the installation"

build-menu-bar:
	@if [ "$$(uname -s)" != "Darwin" ]; then \
		echo "Error: the menu-bar app is only supported on macOS"; \
		exit 1; \
	fi
	mkdir -p "$(BEAR_MENU_BAR_APP)/Contents/MacOS"
	mkdir -p "$(BEAR_MENU_BAR_APP)/Contents/Resources"
	mkdir -p "$(BEAR_MENU_BAR_ICONSET)"
	cd $(BEAR_CLI_WORK_DIR) && go build -trimpath -o "$(BEAR_MENU_BAR_APP)/Contents/MacOS/$(BEAR_MENU_BAR_BINARY)" ./cmd/bear-menu-bar
	cd $(BEAR_CLI_WORK_DIR) && go build -trimpath -o "$(BEAR_MENU_BAR_APP)/Contents/Resources/bear" ./cmd/bear
	cp packaging/macos/Info.plist "$(BEAR_MENU_BAR_APP)/Contents/Info.plist"
	sips -z 16 16 "$(BEAR_MENU_BAR_ICON_SOURCE)" --out "$(BEAR_MENU_BAR_ICONSET)/icon_16x16.png"
	sips -z 32 32 "$(BEAR_MENU_BAR_ICON_SOURCE)" --out "$(BEAR_MENU_BAR_ICONSET)/icon_16x16@2x.png"
	sips -z 32 32 "$(BEAR_MENU_BAR_ICON_SOURCE)" --out "$(BEAR_MENU_BAR_ICONSET)/icon_32x32.png"
	sips -z 64 64 "$(BEAR_MENU_BAR_ICON_SOURCE)" --out "$(BEAR_MENU_BAR_ICONSET)/icon_32x32@2x.png"
	sips -z 128 128 "$(BEAR_MENU_BAR_ICON_SOURCE)" --out "$(BEAR_MENU_BAR_ICONSET)/icon_128x128.png"
	sips -z 256 256 "$(BEAR_MENU_BAR_ICON_SOURCE)" --out "$(BEAR_MENU_BAR_ICONSET)/icon_128x128@2x.png"
	sips -z 256 256 "$(BEAR_MENU_BAR_ICON_SOURCE)" --out "$(BEAR_MENU_BAR_ICONSET)/icon_256x256.png"
	sips -z 512 512 "$(BEAR_MENU_BAR_ICON_SOURCE)" --out "$(BEAR_MENU_BAR_ICONSET)/icon_256x256@2x.png"
	sips -z 512 512 "$(BEAR_MENU_BAR_ICON_SOURCE)" --out "$(BEAR_MENU_BAR_ICONSET)/icon_512x512.png"
	sips -z 1024 1024 "$(BEAR_MENU_BAR_ICON_SOURCE)" --out "$(BEAR_MENU_BAR_ICONSET)/icon_512x512@2x.png"
	go run ./packaging/macos/create_icns.go "$(BEAR_MENU_BAR_ICONSET)" "$(BEAR_MENU_BAR_APP)/Contents/Resources/BearIcon.icns"
	codesign --force --deep --sign - "$(BEAR_MENU_BAR_APP)"
	@echo "Built $(BEAR_MENU_BAR_APP)"

install-menu-bar: build-menu-bar
	mkdir -p "$(BEAR_MENU_BAR_INSTALL_DIR)"
	ditto "$(BEAR_MENU_BAR_APP)" "$(BEAR_MENU_BAR_INSTALL_DIR)/$(BEAR_MENU_BAR_NAME).app"
	@echo "Installed $(BEAR_MENU_BAR_NAME).app to $(BEAR_MENU_BAR_INSTALL_DIR)"

deploy-menu-bar: build-menu-bar
	sudo ditto "$(BEAR_MENU_BAR_APP)" "$(BEAR_MENU_BAR_DEPLOY_DIR)/$(BEAR_MENU_BAR_NAME).app"
	@echo "Deployed $(BEAR_MENU_BAR_NAME).app to $(BEAR_MENU_BAR_DEPLOY_DIR)"

run-menu-bar: build-menu-bar
	open "$(BEAR_MENU_BAR_APP)"

clean:
	rm -f $(BEAR_CLI_OUTPUT)

clean-menu-bar:
	rm -rf "$(BEAR_MENU_BAR_APP)"
	rm -rf "$(BEAR_MENU_BAR_ICONSET)"

reinstall: clean install

reinstall-menu-bar: clean-menu-bar install-menu-bar
