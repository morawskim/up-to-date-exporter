package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
	"up-to-date-exporter/adapter/dockerimage"
	"up-to-date-exporter/adapter/githubrelease"
	"up-to-date-exporter/adapter/githubtag"
	"up-to-date-exporter/adapter/quayimage"
	"up-to-date-exporter/config"

	"github.com/alecthomas/kingpin"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	//nolint: gochecknoglobals
	bind = kingpin.Flag("bind", "addr to bind the server").Default(":9333").String()
	//nolint: gochecknoglobals
	debug = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	//nolint: gochecknoglobals
	configFile = kingpin.Flag("config.file", "config file").Default("config.yaml").ExistingFile()
	version    = "dev"
)

func initTrace(ctx context.Context) (*sdktrace.TracerProvider, error) {
	//res, err := resource.New(
	//	ctx,
	//	resource.WithFromEnv(),
	//	resource.WithProcess(),
	//	resource.WithHost(),
	//	resource.WithTelemetrySDK(),
	//	resource.WithAttributes(
	//		semconv.ServiceName("up-to-date-exporter"),
	//		semconv.ServiceVersion(version),
	//	),
	//)
	//
	//tp := sdktrace.NewTracerProvider(
	//	sdktrace.WithBatcher(traceExporter),
	//	sdktrace.WithResource(res),
	//)

	//exporter, err := stdout.New(stdout.WithPrettyPrint())
	exporter, err := otlptrace.New(ctx, otlptracehttp.NewClient())
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, err
}

func main() {
	kingpin.Version("up-to-date-exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	logger.Info("starting up-to-date-exporter")

	if *debug {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)

		logger.Debug("enabled debug mode")
	}

	ctx := context.Background()
	tp, err := initTrace(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create trace exporter: %v", err))
		panic(err)
	}
	//provisioning
	http.DefaultClient.Transport = otelhttp.NewTransport(http.DefaultTransport)

	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("shutdown tracer: %v", err))
		}
	}()
	otel.SetTracerProvider(tp)

	cacheClient := cache.New(time.Minute*15, time.Minute*15)

	var conf = config.Config{}
	var collectorGitHubReleases, collectorDockerImages, collectorGitHubTags config.ReloadCollectorConfiguration
	var collectorQuayImages config.ReloadCollectorConfiguration

	config.Load(*configFile, &conf, func() {
		collectorGitHubReleases.ReloadConfiguration(&conf)
		collectorDockerImages.ReloadConfiguration(&conf)
		collectorGitHubTags.ReloadConfiguration(&conf)
		collectorQuayImages.ReloadConfiguration(&conf)

		logger.Debug("flushing cache...")
		cacheClient.Flush()
	})

	collectorGitHubReleases = githubrelease.Register("", conf.GithubReleases, cacheClient)
	collectorDockerImages = dockerimage.Register(conf.DockerImages, cacheClient)
	collectorGitHubTags = githubtag.Register(conf.GithubTags, cacheClient)
	collectorQuayImages = quayimage.Register(conf.QuaryImages, cacheClient)
	go refreshData(logger, collectorDockerImages, collectorGitHubTags, collectorQuayImages, collectorGitHubReleases)

	http.Handle("/metrics", otelhttp.NewHandler(promhttp.Handler(), "metrics"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(
			w, `
			<html>
			<head><title>Up-to-date Exporter</title></head>
			<body>
				<h1>Up-to-date Exporter</h1>
				<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>
			`,
		)
	})
	logger.Info(fmt.Sprintf(`listening on %s`, *bind))
	if err := http.ListenAndServe(*bind, nil); err != nil { //nolint:gosec
		logger.Error(fmt.Sprintf("error starting server: %s", err))
		panic(err)
	}
}

func fetchData(logger *slog.Logger, collectors ...config.ReloadCollectorConfiguration) {
	tr := otel.Tracer("")
	ctx, span := tr.Start(context.Background(), "fetchData")
	defer span.End()

	logger.Info("refreshing data")
	for _, c := range collectors {
		c.FetchData(ctx)
	}
}

func refreshData(logger *slog.Logger, collectors ...config.ReloadCollectorConfiguration) {
	fetchData(logger, collectors...)
	for range time.Tick(time.Minute * 5) {
		fetchData(logger, collectors...)
	}
}
