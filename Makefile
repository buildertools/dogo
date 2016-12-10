PWD := $(shell pwd)

clean:
	@rm -f bin/dogo
	@-docker rmi buildertools/dogo:poc buildertools/dogo:latest buildertools/dogo:build-tooling

# Prepare tooling
prepare:
	@docker build -t buildertools/dogo:build-tooling -f tooling.df .

update-deps:
	@docker run --rm -v $(PWD):/go/src/github.com/buildertools/dogo -w /go/src/github.com/buildertools/dogo buildertools/dogo:build-tooling trash -u

update-vendor:
	@docker run --rm -v $(PWD):/go/src/github.com/buildertools/dogo -w /go/src/github.com/buildertools/dogo buildertools/dogo:build-tooling trash

iterate:
	@docker-compose -f iterate.dc kill
	@docker-compose -f iterate.dc rm
	@docker-compose -f iterate.dc up -d

fmt:
	# Formatting
	@docker run --rm \
	  -v $(PWD):/go/src/github.com/buildertools/dogo \
	  -w /go/src/github.com/buildertools/dogo \
	  buildertools/dogo:build-tooling \
	  go fmt

lint:
	# Linting
	@docker run --rm \
	  -v $(PWD):/go/src/github.com/buildertools/dogo \
	  -w /go/src/github.com/buildertools/dogo \
	  buildertools/dogo:build-tooling \
	  golint -set_exit_status

test:
	# Unit testing
	@docker run --rm \
	  -v $(PWD):/go/src/github.com/buildertools/dogo \
	  -v $(PWD)/bin:/go/bin \
	  -w /go/src/github.com/buildertools/dogo \
	  golang:1.7 \
	  go test
	# Test coverage
	@docker run --rm \
	  -v $(PWD):/go/src/github.com/buildertools/dogo \
	  -v $(PWD)/bin:/go/bin \
	  -w /go/src/github.com/buildertools/dogo \
	  golang:1.7 \
	  go test -cover

build: fmt lint test
	# Building binaries
	@docker run --rm \
	  -v $(PWD):/go/src/github.com/buildertools/dogo \
	  -v $(PWD)/bin:/go/bin \
	  -w /go/src/github.com/buildertools/dogo \
	  -e GOOS=linux \
	  -e GOARCH=amd64 \
	  golang:1.7 \
	  go build -o bin/dogo-linux64
	@docker run --rm \
	  -v $(PWD):/go/src/github.com/buildertools/dogo \
	  -v $(PWD)/bin:/go/bin \
	  -w /go/src/github.com/buildertools/dogo \
	  -e GOOS=darwin \
	  -e GOARCH=amd64 \
	  golang:1.7 \
	  go build -o bin/dogo-darwin64

release: build
	@docker build -t buildertools/dogo:latest -f release.df .
	@docker tag buildertools/dogo:latest buildertools/dogo:poc
