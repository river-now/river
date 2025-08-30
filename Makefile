#####################################################################
####### GO
#####################################################################

gotest:
	@go test -race ./...

gotestloud:
	@go test -race -v ./...

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

tsbench:
	@npx vitest bench

tsreset:
	@rm -rf node_modules 2>/dev/null || true
	@find . -path "*/node_modules" -type d -exec rm -rf {} \; 2>/dev/null || true
	@pnpm i
	@cd internal/framework/_typescript/create && pnpm i

tslint:
	@pnpm oxlint

tscheck: tscheck-kit tscheck-fw-client tscheck-fw-react tscheck-fw-solid

tscheck-kit:
	@pnpm tsgo --noEmit --project ./kit/_typescript

tscheck-fw-client:
	@pnpm tsgo --noEmit --project ./internal/framework/_typescript/client

tscheck-fw-react:
	@pnpm tsgo --noEmit --project ./internal/framework/_typescript/react

tscheck-fw-solid:
	@pnpm tsgo --noEmit --project ./internal/framework/_typescript/solid

tscheck-fw-preact:
	@pnpm tsgo --noEmit --project ./internal/framework/_typescript/preact

tsprepforpub: tsreset tstest tslint tscheck

tspublishpre: tsprepforpub
	@npm publish --access public --tag pre
	@cd internal/framework/_typescript/create && npm publish --access public --tag pre

tspublishnonpre: tsprepforpub
	@npm publish --access public
	@cd internal/framework/_typescript/create && npm publish --access public

npmbuild:
	@go run ./internal/scripts/buildts

npmbump:
	@go run ./internal/scripts/npm_bumper

docker-site:
	@docker build -t river-site -f Dockerfile.site .

docker-run-site:
	docker run -d -p $(PORT):$(PORT) -e PORT=$(PORT) river-site
