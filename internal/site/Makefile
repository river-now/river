pre:
	@pnpm i && go mod tidy

dev: pre
	@go run ./__cmd/build --dev

build-prod: pre
	@go run ./__cmd/build
	@echo "\nbuild-prod complete. now run:\n\tmake run-prod port=9090"

# call with `make run-prod port=whatever`
run-prod:
	@PORT=$(port) ./__dist/main

vercel-install:
	@curl -L https://go.dev/dl/go1.24.0.linux-amd64.tar.gz \
		| tar -C /tmp -xz
	@export PATH=/tmp/go/bin:$PATH
	@pnpm i

vercel-build:
	@cd .. && make npmbuild
	@export PATH=/tmp/go/bin:$PATH
	@go run ./__cmd/build --no-binary
