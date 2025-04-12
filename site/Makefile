pre:
	@pnpm i && go mod tidy

tailwind-prod:
	@pnpx @tailwindcss/cli -i ./css/tailwind-input.css -o ./css/tailwind-output.css

tailwind-dev:
	@pnpx @tailwindcss/cli -i ./css/tailwind-input.css -o ./css/tailwind-output.css --watch

serve-dev:
	@go run ./go/cmd/build --dev

dev-inner: tailwind-dev serve-dev

dev: pre
	@make dev-inner -j

build-prod: pre tailwind-prod
	@go run ./go/cmd/build

# call with `make run-prod port=whatever`
run-prod:
	@PORT=$(port) ./go/app/kiruna_dist/main
