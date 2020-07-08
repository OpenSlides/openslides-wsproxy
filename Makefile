build-dev:
	docker build . --target development --tag openslides-wsproxy-dev

run-tests:
	docker build . --target testing --tag openslides-wsproxy-test
	docker run openslides-wsproxy-test
