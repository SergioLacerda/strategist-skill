.PHONY: build test lint cover cover-gate cover-html install-local release clean

build:
	go build -o bin/strategist ./cmd/strategist

test:
	go test -race ./...

lint:
	golangci-lint run ./...

# cover shows per-package coverage (each package measured against itself).
cover:
	@for pkg in internal/stale internal/compile internal/install internal/embed; do \
		echo "=== $$pkg ==="; \
		go test -race -coverprofile=coverage.out -coverpkg=./$$pkg/... ./$$pkg/... 2>/dev/null; \
		go tool cover -func=coverage.out | tail -1; \
	done

# cover-gate fails the build if any internal package is below 90%.
cover-gate:
	@fail=0; \
	for pkg in internal/stale internal/compile internal/install internal/embed; do \
		pct=$$(go test -coverprofile=coverage.out -coverpkg=./$$pkg/... ./$$pkg/... 2>/dev/null \
			| grep -o '[0-9.]*%' | tail -1 | tr -d '%'); \
		printf "%-30s %s%%\n" "$$pkg" "$$pct"; \
		ok=$$(awk -v p="$$pct" 'BEGIN{print (p+0 >= 90)}'); \
		if [ "$$ok" != "1" ]; then echo "  FAIL: $$pct% < 90%"; fail=1; fi; \
	done; \
	exit $$fail

# cover-html opens an HTML report for all internal packages combined.
cover-html:
	go test -race -coverprofile=coverage.out -coverpkg=./internal/... ./internal/... ./tests/...
	go tool cover -html=coverage.out

install-local: build
	install -m 755 bin/strategist ~/.local/bin/strategist

release:
	goreleaser release --clean

clean:
	rm -rf bin/ dist/ coverage.out
