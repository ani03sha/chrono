# chrono — distributed systems clock primitives
#
# All targets are .PHONY because there are no file dependencies; every target
# runs the underlying tool fresh. The cost of a stale target is much higher
# than the cost of redundant work.

.PHONY: all build test test-race coverage bench lint examples proofs clean help

# Default target: the minimum bar for a green build.
all: test lint

# Compile every package. No artifacts produced for libraries; this is a
# correctness check that everything builds.
build:
	go build ./...

# Run every test. Exit non-zero on failure.
test:
	go test ./...

# Run every test under the race detector. Slower, catches data races.
test-race:
	go test -race ./...

# Generate coverage and fail the build if total coverage drops below 90%.
# We parse `go tool cover` output rather than depending on a third-party tool.
coverage:
	go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | awk ' \
			END { \
					gsub("%", "", $$3); \
					if ($$3 + 0 < 90.0) { \
							printf "FAIL: total coverage %.1f%% is below 90%% threshold\n", $$3; \
							exit 1; \
					} \
					printf "OK: total coverage %.1f%%\n", $$3; \
			}'

# Run benchmarks. -run=^$$ disables tests so only Benchmark* functions run.
# -benchmem reports allocations per op, which we want for hot-path validation.
bench:
	go test -bench=. -benchmem -run=^$$ ./benchmarks/... | tee BENCHMARKS.md

# Lint with golangci-lint. Install separately: brew install golangci-lint
lint:
	golangci-lint run ./...

# Build every example program. Each example is its own main package.
examples:
	@for dir in examples/*/; do \
			echo "  building $$dir"; \
			(cd $$dir && go build) || exit 1; \
	done

# Run only the proofs/ scenario tests, with verbose output so the scenario
# narrative is visible in the test log.
proofs:
	go test -v ./proofs/...

# Remove generated files. Safe to re-run.
clean:
	rm -f coverage.out BENCHMARKS.md
	@for dir in examples/*/; do \
			bin="$$dir$$(basename $$dir)"; \
			[ -f "$$bin" ] && rm -f "$$bin" || true; \
	done

help:
	@echo "Targets:"
	@echo "  build      Compile every package (correctness check)"
	@echo "  test       Run all tests"
	@echo "  test-race  Run all tests with the race detector"
	@echo "  coverage   Coverage report; fail if below 90%"
	@echo "  bench      Run benchmarks; write results to BENCHMARKS.md"
	@echo "  lint       Run golangci-lint"
	@echo "  examples   Build every example program"
	@echo "  proofs     Run proofs/ scenario tests (verbose)"
	@echo "  clean      Remove generated files"