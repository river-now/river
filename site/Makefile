pre:
	@pnpm i && go mod tidy

tailwind-prod:
	@pnpx @tailwindcss/cli -i ./css/tailwind-input.css -o ./css/tailwind-output.css

tailwind-dev:
	@pnpx @tailwindcss/cli -i ./css/tailwind-input.css -o ./css/tailwind-output.css --watch

serve-dev: pre
	@go run ./go/cmd/build --dev

dev: tailwind-dev serve-dev

build-prod: pre tailwind-prod
	@go run ./go/cmd/build

# call with `make run-prod port=whatever`
run-prod:
	@PORT=$(port) ./go/app/kiruna_dist/main
