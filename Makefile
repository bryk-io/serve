.PHONY: *
.DEFAULT_GOAL:=help

# Project setup
BINARY_NAME=serve
OWNER=bryk-io
REPO=serve
PROJECT_REPO=github.com/$(OWNER)/$(REPO)
DOCKER_IMAGE=ghcr.io/$(OWNER)/$(BINARY_NAME)
MAINTAINERS='Ben Cessa <ben@pixative.com>'

# State values
GIT_COMMIT_DATE=$(shell TZ=UTC git log -n1 --pretty=format:'%cd' --date='format-local:%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT_HASH=$(shell git log -n1 --pretty=format:'%H')
GIT_TAG=$(shell git describe --tags --always --abbrev=0 | cut -c 1-7)

# Linker tags
# https://golang.org/cmd/link/
LD_FLAGS += -s -w
LD_FLAGS += -X $(PROJECT_REPO)/internal.CoreVersion=$(GIT_TAG:v%=%)
LD_FLAGS += -X $(PROJECT_REPO)/internal.BuildTimestamp=$(GIT_COMMIT_DATE)
LD_FLAGS += -X $(PROJECT_REPO)/internal.BuildCode=$(GIT_COMMIT_HASH)

# For commands that require a specific package path, default to all local
# subdirectories if no value is provided.
pkg?="..."

help:
	@echo "Commands available"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /' | sort

## api-docs: Runs a local Swagger UI server with the API definitions
api-docs:
	docker run -it --rm --platform linux/amd64 -p 8080:8080 -e SWAGGER_JSON=/docs/service_api.swagger.json -v $(shell pwd)/proto/sample/v1/:/docs swaggerapi/swagger-ui
	$(shell open http://localhost:8080)

## bench: Run benchmarks
bench:
	go test -run=XXX -bench=. ./$(pkg)

## build: Build for the current architecture in use, intended for development
build:
	# Build CLI application
	go build -v -ldflags '$(LD_FLAGS)' -o $(BINARY_NAME) ./cli

## build-for: Build the available binaries for the specified 'os' and 'arch'
# make build-for os=linux arch=amd64
build-for:
	CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) \
	go build -v -ldflags '$(LD_FLAGS)' \
	-o $(BINARY_NAME)_$(os)_$(arch)$(suffix) \
	./cli

## ca-roots: Generate the list of valid CA certificates
ca-roots:
	@docker run -dit --rm --name ca-roots debian:stable-slim
	@docker exec --privileged ca-roots sh -c "apt update"
	@docker exec --privileged ca-roots sh -c "apt install -y ca-certificates"
	@docker exec --privileged ca-roots sh -c "cat /etc/ssl/certs/* > /ca-roots.crt"
	@docker cp ca-roots:/ca-roots.crt ca-roots.crt
	@docker stop ca-roots

## deps: Download and compile all dependencies and intermediary products
deps:
	go mod tidy
	go clean

## docker: Build docker image
# https://github.com/opencontainers/image-spec/blob/master/annotations.md
docker:
	make build-for os=linux arch=amd64
	mv $(BINARY_NAME)_linux_amd64 $(BINARY_NAME)
	@-docker rmi $(DOCKER_IMAGE):$(GIT_TAG:v%=%)
	@docker build \
	"--label=org.opencontainers.image.title=$(BINARY_NAME)" \
	"--label=org.opencontainers.image.authors=$(MAINTAINERS)" \
	"--label=org.opencontainers.image.created=$(GIT_COMMIT_DATE)" \
	"--label=org.opencontainers.image.revision=$(GIT_COMMIT_HASH)" \
	"--label=org.opencontainers.image.version=$(GIT_TAG:v%=%)" \
	--rm -t $(DOCKER_IMAGE):$(GIT_TAG:v%=%) .
	@rm $(BINARY_NAME)

## install: Install the binary to GOPATH and keep cached all compiled artifacts
install:
	@go build -v -ldflags '$(LD_FLAGS)' -o ${GOPATH}/bin/$(BINARY_NAME) ./cli

## lint: Static analysis
lint:
	# Code
	golangci-lint run -v ./$(pkg)

## release: Prepare artifacts for a new tagged release
release:
	goreleaser release --skip-validate --skip-publish --rm-dist

## scan-deps: Look for known vulnerabilities in the project dependencies
# https://github.com/sonatype-nexus-community/nancy
scan-deps:
	@go list -mod=readonly -f '{{if not .Indirect}}{{.}}{{end}}' -m all | nancy sleuth --skip-update-check

## scan-secrets: Scan project code for accidentally leaked secrets
scan-secrets:
	@docker run --platform linux/amd64 --rm \
	-v $(shell pwd):/proj \
	dxa4481/trufflehog file:///proj \
	-x .exclude-secrets-scan.txt \
	--regex \
	--entropy false

## test: Run unit tests excluding the vendor dependencies
test:
	go test -race -v -failfast -coverprofile=coverage.report ./...
	go tool cover -html coverage.report -o coverage.html

## updates: List available updates for direct dependencies
# https://github.com/golang/go/wiki/Modules#how-to-upgrade-and-downgrade-dependencies
updates:
	@GOWORK=off go list -u -f '{{if (and (not (or .Main .Indirect)) .Update)}}{{.Path}} [{{.Version}} -> {{.Update.Version}}]{{end}}' -m all 2> /dev/null
