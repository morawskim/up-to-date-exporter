.PHONY: publish-docker-image
publish-docker-image:
	docker build -t morawskim/up-to-date-exporter .
	docker push morawskim/up-to-date-exporter

.PHONY: a
a:
	docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:v1.50.1 golangci-lint run
