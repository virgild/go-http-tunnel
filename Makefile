
GO_FILES := $(shell \
	find . '(' -path '*/.*' -o -path './vendor' ')' -prune \
	-o -name '*.go' -print | cut -b3-)

LINT_IGNORE := "/id/\|/tunnelmock/\|/vendor/"

all: clean check test

.PHONY: clean
clean:
	@rm -rf build
	@go clean -r

.PHONY: fmt
fmt:
	@go fmt ./...

.PHONY: check
check: .check-fmt .check-vet .check-lint .check-ineffassign .check-static .check-misspell

.PHONY: .check-fmt
.check-fmt:
	$(eval FMT_LOG := $(shell mktemp -t gofmt.XXXXX))
	@cat /dev/null > $(FMT_LOG)
	@gofmt -e -s -l -d $(GO_FILES) > $(FMT_LOG) || true
	@[ ! -s "$(FMT_LOG)" ] || (echo "$@ failed:" | cat - $(FMT_LOG) && false)

.PHONY: .check-vet
.check-vet:
	@go vet ./...

.PHONY: .check-lint
.check-lint:
	$(eval LINT_LOG := $(shell mktemp -t golint.XXXXX))
	@cat /dev/null > $(LINT_LOG)
	@$(foreach pkg, $(GO_FILES), golint $(pkg | grep -v $LINT_IGNORE) >> $(LINT_LOG) || true;)
	@[ ! -s "$(LINT_LOG)" ] || (echo "$@ failed:" | cat - $(LINT_LOG) && false)


.PHONY: .check-ineffassign
.check-ineffassign:
	@ineffassign ./

.PHONY: .check-misspell
.check-misspell:
	@misspell ./...

.PHONY: .check-mega
.check-static:
	@staticcheck -checks ['SA1006','ST1005'] ./...

.PHONY: test
test:
	@echo "==> Running tests (race)..."
	@go test -cover -race ./...

.PHONY: get-tools
get-tools:
	@echo "==> Installing tools..."
	@GO111MODULE=off go get -u golang.org/x/lint/golint
	@GO111MODULE=off go get -u github.com/golang/mock/gomock

	@GO111MODULE=off go get -u github.com/client9/misspell/cmd/misspell
	@GO111MODULE=off go get -u github.com/gordonklaus/ineffassign
	@GO111MODULE=off go get -u github.com/tcnksm/ghr
	@GO111MODULE=off go get -u honnef.co/go/tools/cmd/staticcheck

OUTPUT_DIR = build
GIT_COMMIT = $(shell git describe --always)

$(OUTPUT_DIR):
	@mkdir $(OUTPUT_DIR)

.PHONY: binaries
binaries: $(OUTPUT_DIR)/tunneld $(OUTPUT_DIR)/tunneld-linux

$(OUTPUT_DIR)/tunneld: $(OUTPUT_DIR)
	@CGO_ENABLED=0 go build -ldflags "-w -X main.version=$(GIT_COMMIT)" \
  		-o "$(OUTPUT_DIR)/tunneld" \
		./cmd/tunneld

.PHONY: tunneld-linux
$(OUTPUT_DIR)/tunneld-linux: $(OUTPUT_DIR)
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-w -X main.version=$(GIT_COMMIT)" \
		-o "$(OUTPUT_DIR)/tunneld-linux" \
		./cmd/tunneld


GCLOUD_PROJECT := $(shell gcloud config get-value project)

.PHONY: docker
docker:
	docker build -t us.gcr.io/$(GCLOUD_PROJECT)/tunneld:latest .

.PHONY: docker-push
docker-push:
	docker push us.gcr.io/$(GCLOUD_PROJECT)/tunneld:latest
