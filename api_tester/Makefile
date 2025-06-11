
BINARY_NAME=kadrion

GO_CMD=go
INSTALL_PATH=/usr/local/bin
.PHONY: build
build:
	$(GO_CMD) build -o $(BINARY_NAME) ./cmd

.PHONY: run
run: build
	./$(BINARY_NAME)

.PHONY: delete
delete:
	rm -f $(BINARY_NAME)

.PHONY: clean
clean: delete
	$(GO_CMD) clean

.PHONY: install
install: build
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)
	if ! grep -q 'export PATH=$(INSTALL_PATH):$$PATH' ~/.bashrc; then \
	    echo 'export PATH=$(INSTALL_PATH):$$PATH' >> ~/.bashrc; \
	fi
	. ~/.bashrc


.PHONY: remove
remove:
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
