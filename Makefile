WAILS ?= $(shell { command -v wails 2>/dev/null || { gobin="$$(go env GOBIN)"; [ -x "$$gobin/wails" ] && echo "$$gobin/wails" || echo "$$(go env GOPATH)/bin/wails"; }; })

UI_TAG :=
ifeq ($(shell uname),Linux)
UI_TAG = webkit2_41
endif

TAGS := $(if $(UI_TAG),-tags $(UI_TAG),)

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: dev
dev:
	$(WAILS) dev $(TAGS)

.PHONY: build
build:
	$(WAILS) build -clean $(TAGS)

.PHONY: run
run: build
	./build/bin/vidctl

.PHONY: test
test:
	go test ./...

.PHONY: check
check:
	go vet ./...
	go test ./...
	test -z "$$(gofmt -l .)"
	npm --prefix frontend run check

.PHONY: smoke
smoke:
	VIDCTL_SMOKE="$${VIDCTL_SMOKE:?defina VIDCTL_SMOKE=/caminho/para/video.mp4}" go test -run TestSmokeReal -v .

.PHONY: clean
clean:
	rm -rf build/bin frontend/dist
	go clean