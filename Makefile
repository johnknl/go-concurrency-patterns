TOOLS := -modfile=tools.mod

.PHONY: test lint headers vuln check docs docs-check docs-serve mutation mutation-operators mutation-concurrency

test:
	go vet ./...
	go test -race ./...

lint:
	go tool $(TOOLS) golangci-lint run ./...

# Adds the license header from LICENSE to Go files that lack it.
headers:
	go tool $(TOOLS) addlicense -f LICENSE $$(git ls-files '*.go')

vuln:
	go tool $(TOOLS) govulncheck ./...

check: test lint vuln docs-check

# Mutation testing of the pattern code; the report is in .tmp/mutation.
# Mutation testing of the pattern code. gremlins mutates operators (arithmetic, comparisons, conditions);
# tools/mutate mutates the concurrency mechanism (close, select cases, locks, waits, cancels, buffers).
MUTATE_DIRS := $(wildcard patterns/* antipatterns/* stdlib/*)

mutation: mutation-operators mutation-concurrency

mutation-operators:
	mkdir -p .tmp/mutation
	go tool $(TOOLS) gremlins unleash --exclude-files 'tools/' .

mutation-concurrency:
	go tool mutate -timeout 20s $(MUTATE_DIRS)

docs:
	go tool docsync sync

docs-check:
	go tool docsync check

docs-serve: docs
	docker run --rm -it -p 8000:8000 -v "$(CURDIR):/docs" squidfunk/mkdocs-material:9
