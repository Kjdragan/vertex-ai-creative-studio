// otel.go — shared OTel setup for MCPs (defaults to HTTP exporter)
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc/credentials"
)

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseHeaders() map[string]string {
	// Prefer OTEL headers if present, else synthesize from ARIZE_*
	h := firstNonEmpty(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS"), os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"))
	headers := map[string]string{}
	if h != "" {
		for _, kv := range strings.Split(h, ",") {
			kv = strings.TrimSpace(kv)
			if kv == "" || !strings.Contains(kv, "=") {
				continue
			}
			parts := strings.SplitN(kv, "=", 2)
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	if _, ok := headers["space_id"]; !ok {
		if sid := strings.TrimSpace(os.Getenv("ARIZE_SPACE_ID")); sid != "" {
			headers["space_id"] = sid
		}
	}
	if _, ok := headers["api_key"]; !ok {
		if key := strings.TrimSpace(os.Getenv("ARIZE_API_KEY")); key != "" {
			headers["api_key"] = key
		}
	}
	// non-essential niceties for debugging on the Arize side
	if _, ok := headers["arize-interface"]; !ok {
		if ui := strings.TrimSpace(os.Getenv("ARIZE_INTERFACE")); ui != "" {
			headers["arize-interface"] = ui
		}
	}
	if _, ok := headers["arize-project"]; !ok {
		if pn := strings.TrimSpace(os.Getenv("ARIZE_PROJECT_NAME")); pn != "" {
			headers["arize-project"] = pn
		}
	}
	return headers
}

func samplerRatio() float64 {
	arg := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG"))
	if arg == "" {
		return 1.0
	}
	// best-effort parse
	var f float64 = 1.0
	fmt.Sscanf(arg, "%f", &f)
	return f
}

func scrubHeaders(h map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range h {
		if k == "api_key" || strings.Contains(strings.ToLower(k), "auth") {
			out[k] = "****"
		} else if k == "space_id" {
			if len(v) > 6 {
				out[k] = v[:6] + "…"
			} else {
				out[k] = "****"
			}
		} else {
			out[k] = v
		}
	}
	return out
}

// initOpenTelemetry is called by main() in each MCP.
func initOpenTelemetry(serviceName, serviceVersion string) (func(context.Context) error, error) {
	// Decide transport (default HTTP)
	proto := strings.ToLower(firstNonEmpty(
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"),
		os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"),
	))
	if proto == "" || strings.HasPrefix(proto, "http") {
		proto = "http/protobuf"
	}

	endpoint := firstNonEmpty(
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"),
		os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	)

	headers := parseHeaders()
	if headers["space_id"] == "" || headers["api_key"] == "" {
		return nil, fmt.Errorf("missing ARIZE creds: space_id or api_key")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var (
		exp *otlptrace.Exporter
		err error
	)

	switch proto {
	case "http", "http/protobuf":
		// Normalize endpoint to URL; accept bare host or full URL, with/without path.
		u, _ := url.Parse(endpoint)
		if endpoint == "" || u.Host == "" {
			u = &url.URL{Scheme: "https", Host: "otlp.arize.com", Path: "/v1/traces"}
		} else {
			if u.Scheme == "" {
				u.Scheme = "https"
			}
			if u.Path == "" {
				u.Path = "/v1/traces"
			}
		}
		client := otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(u.Host),
			otlptracehttp.WithURLPath(strings.TrimPrefix(u.Path, "/")),
			otlptracehttp.WithTLSClientConfig(&tls.Config{}),
			otlptracehttp.WithHeaders(headers),
		)
		exp, err = otlptrace.New(ctx, client)

	default: // grpc
		hostPort := endpoint
		if hostPort == "" || strings.HasPrefix(hostPort, "http") {
			hostPort = "otlp.arize.com:443"
		}
		client := otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(hostPort),
			otlptracegrpc.WithTLSCredentials(credentials.NewTLS(&tls.Config{})),
			otlptracegrpc.WithHeaders(headers),
		)
		exp, err = otlptrace.New(ctx, client)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	res, _ := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			attribute.String("arize.project", firstNonEmpty(os.Getenv("ARIZE_PROJECT_NAME"), "genmedia-adk")),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exp)),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(samplerRatio())),
	)
	otel.SetTracerProvider(tp)

	fmt.Printf("🔭 OTEL exporter ready — transport=%s, endpoint=%s, headers=%v\n", proto, firstNonEmpty(endpoint, "(default)"), scrubHeaders(headers))
	return tp.Shutdown, nil
}
