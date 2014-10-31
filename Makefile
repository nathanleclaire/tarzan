DOCKER_IMAGE := nathanleclaire/tarzan
DOCKER_CONTAINER := tarzan-build
DOCKER_SRC_PATH := /go/src/github.com/nathanleclaire/tarzan

default: dockerbuild

dockerbuild: clean
	docker build -t $(DOCKER_IMAGE) .
	docker run --name $(DOCKER_CONTAINER) --entrypoint true $(DOCKER_IMAGE)
	docker cp $(DOCKER_CONTAINER):$(DOCKER_SRC_PATH)/tarzan .
	docker rm $(DOCKER_CONTAINER)

cleanbinary:
	rm -f tarzan

cleancontainers:
	docker rm $(DOCKER_CONTAINER) $(DOCKER_CONTAINER)-deps $(DOCKER_CONTAINER)-test 2>/dev/null || true

deps: cleancontainers
	docker run --name $(DOCKER_CONTAINER)-deps \
		-v $(shell pwd):$(DOCKER_SRC_PATH) \
		$(DOCKER_IMAGE) sh -c "go get github.com/tools/godep && pwd && go get ./... && godep save"
	docker rm $(DOCKER_CONTAINER)-deps 2>/dev/null || true

test: dockerbuild
	docker run --name $(DOCKER_CONTAINER)-test --entrypoint sh $(DOCKER_IMAGE) -c 'go test'
	docker rm $(DOCKER_CONTAINER)-test 2>/dev/null || true

clean: cleanbinary cleancontainers
