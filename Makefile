BINARY := ultui
PKG    := ./...
EXAMPLE ?= examples/dashboard.json

.PHONY: build test race test-one vet fmt tidy run validate render compiled clean

build: ## compile the ultui runtime into bin/
	go build -o bin/$(BINARY) .

test: ## run the whole test suite
	go test $(PKG)

race: ## run the suite under the race detector (shell widget uses goroutines)
	go test -race $(PKG)

## run a single test: make test-one T=TestFocusTraversalWraps
test-one:
	go test ./... -run '$(T)' -v

vet:
	go vet $(PKG)

fmt:
	gofmt -w .

tidy:
	go mod tidy

run: build ## run a config (make run EXAMPLE=examples/hello.json)
	./bin/$(BINARY) run $(EXAMPLE)

validate: build
	./bin/$(BINARY) validate $(EXAMPLE)

render: build ## print one headless frame (make render EXAMPLE=... SIZE=120x40)
	./bin/$(BINARY) render $(EXAMPLE) $(or $(SIZE),100x30)

compiled: ## build the go:embed static-linked demo
	go build -o bin/baked ./examples/compiled

clean:
	rm -rf bin
