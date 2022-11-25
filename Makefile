.PHONY: all help build build-image push-image image

IMAGE_REPO ?= spirkaa
IMAGE_NAME ?= vault-bootstrap
IMAGE_TAG  ?= $$(git log --abbrev-commit --format=%h -s | head -n 1)

all: help

help:
	@echo "    build                          Build app"
	@echo "    build-image                    Build docker image"
	@echo "    push-image                     Push docker image to repo"
	@echo "    image                          build + push"

build:
	@go build -v -o ${IMAGE_NAME}

build-image:
	@DOCKER_BUILDKIT=1 docker build \
		--tag $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) \
		--tag $(IMAGE_REPO)/$(IMAGE_NAME):latest \
		-f local.Dockerfile \
		.

push-image: build-image
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME):latest

image: build-image push-image
