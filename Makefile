build-dev:
	docker build . -f docker/Dockerfile.dev --tag openslides-wsproxy-dev

run-tests:
	docker build . -f docker/Dockerfile.test --tag openslides-wsproxy-test
	docker run openslides-wsproxy-test
