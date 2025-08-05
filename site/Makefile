pre:
	@pnpm i && go mod tidy

tailwind-prod:
	@pnpx @tailwindcss/cli \
		-i ./frontend/css/tailwind-input.css \
		-o ./frontend/css/tailwind-output.css

tailwind-dev:
	@pnpx @tailwindcss/cli \
		-i ./frontend/css/tailwind-input.css \
		-o ./frontend/css/tailwind-output.css \
		--watch 2>&1 | sed 's/^/[tailwind-cli] /'

serve-dev:
	@go run ./__cmd/build --dev

dev-inner: tailwind-dev serve-dev

dev: pre
	@make dev-inner -j

build-prod: pre tailwind-prod
	@go run ./__cmd/build
	@echo "\nbuild-prod complete. now run:\n\tmake run-prod port=9090"

# call with `make run-prod port=whatever`
run-prod:
	@PORT=$(port) ./__dist/main

cf-write-asset-headers-file:
	@echo "/*\n\tCache-Control: public, max-age=31536000, immutable" > __dist/static/assets/public/_headers

cf-deploy: build-prod
	@make cf-write-asset-headers-file && pnpm wrangler deploy
