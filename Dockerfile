FROM cgr.dev/chainguard/go:1.19 as build
USER root

WORKDIR /work
COPY . .
RUN CGO_ENABLED=0 go build -o up-to-date-exporter .

FROM cgr.dev/chainguard/static:latest
COPY --from=build /work/up-to-date-exporter /up-to-date-exporter
ENTRYPOINT ["/up-to-date-exporter"]
