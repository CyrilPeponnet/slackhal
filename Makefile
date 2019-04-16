PROJECT_NAME := slackhal
PROJECT_VERSION ?= v0.0.0-dev
DOCKER_REGISTRY ?= gcr.io/aporetodev
DOCKER_IMAGE_NAME?=$(PROJECT_NAME)
DOCKER_IMAGE_TAG?=$(PROJECT_VERSION)

lint:
	# --enable=unparam
	golangci-lint run \
		--disable-all \
		--exclude-use-default=false \
		--enable=errcheck \
		--enable=goimports \
		--enable=ineffassign \
		--enable=golint \
		--enable=unused \
		--enable=structcheck \
		--enable=staticcheck \
		--enable=varcheck \
		--enable=deadcode \
		--enable=unconvert \
		--enable=misspell \
		--enable=prealloc \
		--enable=nakedret \
		./...

test: lint
	@go test ./... -race -cover -covermode=atomic

build: test
	go build

build_linux: test
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

container: build_linux
	cp slackhal docker/
	cd docker && docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

$(DOCKER_REGISTRY):
	cd docker \
		&& docker tag $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) $@/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) \
		&& docker push $@/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)

push: container $(DOCKER_REGISTRY)
