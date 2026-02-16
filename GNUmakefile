default: build

build:
	go build -o terraform-provider-onelogin

install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/spbsoluble/onelogin/0.0.1/$(shell go env GOOS)_$(shell go env GOARCH)
	cp terraform-provider-onelogin ~/.terraform.d/plugins/registry.terraform.io/spbsoluble/onelogin/0.0.1/$(shell go env GOOS)_$(shell go env GOARCH)/

test:
	go test ./... -v -count=1

testacc:
	TF_ACC=1 go test ./... -v -count=1 -timeout 120m

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

generate:
	go generate ./...

tfdocs:
	tfplugindocs generate
	terraform fmt -recursive ./examples/

.PHONY: build install test testacc lint fmt generate tfdocs
