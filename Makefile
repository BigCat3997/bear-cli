BEAR_CLI_OUTPUT ?= bear
BEAR_CLI_TARGET_DIR ?= /usr/local/bin
BEAR_CLI_WORK_DIR ?= ./

build:
	cd $(BEAR_CLI_WORK_DIR) && go build -o $(BEAR_CLI_OUTPUT)
	ls -la

install: build
	cp $(BEAR_CLI_OUTPUT) $(BEAR_CLI_TARGET_DIR)
	@echo "Installed $(BEAR_CLI_OUTPUT) to $(BEAR_CLI_TARGET_DIR)"

clean:
	rm -f $(BEAR_CLI_OUTPUT)

reinstall: clean install