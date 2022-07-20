export GO111MODULE := on

.DEFAULT_GOAL := build

.PHONY: build
build: test
	go build ./...

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: test
test:
	go test -coverprofile cover.out -cover -race ./... && go tool cover -func cover.out

.PHONY: bustest
bustest: gofmt
	cd bus/testenv && make

.PHONY: show-gomod-version
show-gomod-version:
	@echo This would be the version for your go.mod
	@echo "v0.0.0-"`TZ=utc git log -1 --pretty=format:%cd --date=format-local:"%Y%m%d%H%M%S" HEAD`"-"`git rev-parse --short=12 HEAD`

.PHONY: gofmt
gofmt:
	GO111MODULE=off go fmt ./...

.PHONY: testenv
testenv:
	@cd bus/testenv && make --no-print-directory

.PHONY: mocks
mocks:
	docker run --user $$(id -u):$$(id -g) --rm -w /work -v ${PWD}:/work vektra/mockery:v2.14.0 --name testClient --dir /work/pkg/genericcli --output /work/pkg/genericcli --filename generic_mock_test.go --testonly --inpackage
