
ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: build build-docker build-extension clean

build: clean build-docker build-extension run-test-instance

build-docker:
	docker build \
		--target build-base \
		-t pg-extension-build-base \
		.
	docker build \
		-t pg-extension-builder \
		.

build-extension:
	docker run \
		--rm \
		-v ${ROOT_DIR}/dist:/dist \
		pg-extension-builder

run-test-instance:
	docker build -t pg-extension-test -f postgres.dockerfile .
	docker run --rm -e POSTGRES_PASSWORD=password -i -v ${ROOT_DIR}/dist:/dist --name pg_test -p 5432:5432 pg-extension-test
	docker rmi pg-extension-test

clean:
	docker rmi pg-extension-builder 2>/dev/null || echo 'No builder image to remove'
	rm -rf ./dist/extension
	rm -rf ./dist/lib
	