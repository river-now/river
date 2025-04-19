pre:
	@pnpm i && go mod tidy

tailwind-prod:
	@pnpx @tailwindcss/cli -i ./frontend/css/tailwind-input.css -o ./frontend/css/tailwind-output.css

tailwind-dev:
	@pnpx @tailwindcss/cli -i ./frontend/css/tailwind-input.css -o ./frontend/css/tailwind-output.css --watch

serve-dev:
	@go run ./backend/cmd/build --dev

dev-inner: tailwind-dev serve-dev

dev: pre
	@make dev-inner -j

build-prod: pre tailwind-prod
	@go run ./backend/cmd/build

# call with `make run-prod port=whatever`
run-prod:
	@PORT=$(PORT) ./__dist/main
