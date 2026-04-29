BINARY_NAME=fillarpd
PREFIX=/usr/local
BIN_DIR=$(PREFIX)/bin
SYSTEMD_DIR=/etc/systemd/system
CONFIG_DIR=/etc/default
CONFIG_FILE=$(CONFIG_DIR)/$(BINARY_NAME)
SERVICE_FILE=$(BINARY_NAME).service
GO_FLAGS=-ldflags="-s -w"
export CGO_ENABLED=1

.PHONY: all build install uninstall clean

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	go build $(GO_FLAGS) -o $(BINARY_NAME) ./cmd/fillarpd/main.go

install: build
	@echo "Installing binary to $(BIN_DIR)..."
	install -d $(DESTDIR)$(BIN_DIR)
	install -m 755 $(BINARY_NAME) $(DESTDIR)$(BIN_DIR)/$(BINARY_NAME)

	@echo "Installing default config to $(CONFIG_DIR)..."
	install -d $(DESTDIR)$(CONFIG_DIR)
	@if [ ! -f $(DESTDIR)$(CONFIG_FILE) ]; then \
		echo "INTERFACE=placeholder\nSOURCE_IP=192.168.1.1\nNETWORK=192.168.1.1/24\nSWEEP_INTERVAL=60\nTHREADS=24" > $(DESTDIR)$(CONFIG_FILE); \
	else \
		echo "Config file already exists, skipping..."; \
	fi
	install -d $(DESTDIR)$(SYSTEMD_DIR)
	@echo "Configuring and installing systemd service..."
	@sed -e "s|ExecStart=.*|ExecStart=$(BIN_DIR)/$(BINARY_NAME)|g" \
	     -e "s|EnvironmentFile=.*|EnvironmentFile=$(CONFIG_FILE)|g" \
	     $(SERVICE_FILE) > $(DESTDIR)$(SYSTEMD_DIR)/$(SERVICE_FILE)
	
uninstall:
	@echo "Removing $(BINARY_NAME)..."
	rm -f $(DESTDIR)$(BIN_DIR)/$(BINARY_NAME)
	rm -f $(DESTDIR)$(SYSTEMD_DIR)/$(SERVICE_FILE)
	@echo "Note: Configuration at $(CONFIG_FILE) was not removed."
	systemctl daemon-reload

clean:
	rm -f $(BINARY_NAME)