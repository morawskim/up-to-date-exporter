.PHONY: publish-docker-image
publish-docker-image:
	docker build -t morawskim/up-to-date-exporter .
	docker push morawskim/up-to-date-exporter
