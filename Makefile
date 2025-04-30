include .env

#####################################################################
####### GO
#####################################################################

gotest:
	@go test ./...

gotestloud:
	@go test -v ./...

gobump: gotest
	@go run ./internal/scripts/bumper

# call with `make gobench pkg=./kit/mux` (or whatever)
gobench:
	@go test -bench=. $(pkg)

#####################################################################
####### TS
#####################################################################

tstest:
	@pnpm vitest run

tstestwatch:
	@pnpm vitest

tsreset:
	@rm -rf node_modules 2>/dev/null || true
	@find . -path "*/node_modules" -type d -exec rm -rf {} \; 2>/dev/null || true
	@pnpm i

tslint:
	@pnpm biome check .

tscheck: tscheck-kit tscheck-fw-client tscheck-fw-react tscheck-fw-solid

tscheck-kit:
	@pnpm tsc --noEmit --project ./kit/_typescript

tscheck-fw-client:
	@pnpm tsc --noEmit --project ./internal/framework/_typescript/client

tscheck-fw-react:
	@pnpm tsc --noEmit --project ./internal/framework/_typescript/react

tscheck-fw-solid:
	@pnpm tsc --noEmit --project ./internal/framework/_typescript/solid

tsprepforpub: tsreset tstest tslint tscheck

tspublishpre: tsprepforpub
	@npm publish --access public --tag pre

tspublishnonpre: tsprepforpub
	@npm publish --access public

npmbuild:
	@go run ./internal/scripts/buildts

npmbump:
	@go run ./internal/scripts/npm_bumper

docker-site:
	@docker build -t river-site -f Dockerfile.site .

docker-run-site:
	docker run -d -p $(PORT):$(PORT) -e PORT=$(PORT) river-site

ghcr-login:
	@echo $(GITHUB_TOKEN) | docker login ghcr.io -u river-now --password-stdin

ghcr-build:
	@docker buildx build --platform linux/amd64 -t ghcr.io/river-now/river:latest -f Dockerfile.site . --load

ghcr-push:
	@docker push ghcr.io/river-now/river:latest
