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
