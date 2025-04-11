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
	@docker build -t river-site .

docker-run-site:
	docker run -d -p $(port):$(port) -e PORT=$(port) river-site
