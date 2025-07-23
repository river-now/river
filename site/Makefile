pre:
	@pnpm i && go mod tidy

dev: pre
	@go run ./__cmd/build --dev

build-prod: pre
	@go run ./__cmd/build

# call with `make run-prod port=whatever`
run-prod:
	@PORT=$(PORT) ./__dist/main

cf-write-asset-headers-file:
	@echo "/*\n\tCache-Control: public, max-age=31536000, immutable" > __dist/static/assets/public/_headers

cf-deploy: build-prod
	@make cf-write-asset-headers-file && pnpm wrangler deploy
