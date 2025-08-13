pre:
	@pnpm i && go mod tidy

dev: pre
	@go run ./__cmd/build --dev

build: pre
	@go run ./__cmd/build
	@echo "\nbuild complete. now run:\n\tmake run port=9090"

# call with `make run port=whatever`
run:
	@PORT=$(port) ./app/__dist/main

vercel-install:
	@curl -L https://go.dev/dl/go1.24.0.linux-amd64.tar.gz | tar -C /tmp -xz && \
		export PATH=/tmp/go/bin:$$PATH && \
		pnpm i

vercel-build:
	@export PATH=/tmp/go/bin:$$PATH && \
		go run ./__cmd/build --no-binary
