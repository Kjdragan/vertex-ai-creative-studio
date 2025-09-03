// HTTP-first OpenTelemetry setup for MCP servers.
// Falls back to ARIZE_* if OTEL headers aren't set.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
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

func envBool(keys ...string) bool {
	for _, k := range keys {
		v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
		if v == "1" || v == "true" || v == "yes" {
			return true
		}
	}
	return false
}

func firstNonEmpty(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func parseHeaders() map[string]string {
	// Prefer OTEL_* headers
	h := firstNonEmpty("OTEL_EXPORTER_OTLP_TRACES_HEADERS", "OTEL_EXPORTER_OTLP_HEADERS")
	if h != "" {
		out := map[string]string{}
		for _, kv := range strings.Split(h, ",") {
			kv = strings.TrimSpace(kv)
			if kv == "" {
				continue
			}
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	// Fallback to ARIZE_*
	sid := strings.TrimSpace(os.Getenv("ARIZE_SPACE_ID"))
	key := strings.TrimSpace(os.Getenv("ARIZE_API_KEY"))
	if sid != "" && key != "" {
		return map[string]string{
			"space_id":          sid,
			"api_key":           key,
			"arize-space-id":    sid, // belt & suspenders
			"authorization":     key, // some collectors accept this
			"arize-interface":   "go-mcp",
		}
	}
	return map[string]string{}
}

type httpTarget struct {
	endpoint string // host[:port] (no scheme, no path)
	path     string // defaults to /v1/traces
	insecure bool
}

func parseHTTPSettings() httpTarget {
	raw := firstNonEmpty("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT")
	insecure := envBool("OTEL_EXPORTER_OTLP_TRACES_INSECURE", "OTEL_EXPORTER_OTLP_INSECURE")
	def := httpTarget{endpoint: "otlp.arize.com", path: "/v1/traces", insecure: false}

	if raw == "" {
		return def
	}

	// Accept full URLs (https://.../v1/traces) or just host[:port]
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, err := url.Parse(raw)
		if err != nil {
			return def
		}
		host := u.Host
		p := u.Path
		if p == "" || p == "/" {
			p = "/v1/traces"
		}
		// http scheme implies insecure if user didn't force TLS
		if u.Scheme == "http" {
			insecure = true
		}
		return httpTarget{endpoint: host, path: p, insecure: insecure}
	}

	// No scheme; treat as host[:port]
	host := raw
	if !strings.Contains(host, ":") {
		// default https port for HTTP exporter unless insecure->4318 common local
		if insecure {
			host = net.JoinHostPort(host, "4318")
		} else {
			host = net.JoinHostPort(host, "443")
		}
	}
	return httpTarget{endpoint: host, path: "/v1/traces", insecure: insecure}
}

func buildHTTPExporter(ctx context.Context, hdr map[string]string) (*otlptrace.Exporter, error) {
	t := parseHTTPSettings()
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(t.endpoint),
		otlptracehttp.WithURLPath(t.path),
		otlptracehttp.WithHeaders(hdr),
		otlptracehttp.WithTimeout(10 * time.Second),
	}
	if t.insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	return otlptracehttp.New(ctx, opts...)
}

func buildGRPCExporter(ctx context.Context, hdr map[string]string) (*otlptrace.Exporter, error) {
	// gRPC endpoint envs may be host[:port] or URL; normalize to host:port
	raw := firstNonEmpty("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT")
	hostport := raw
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		u, err := url.Parse(raw)
		if err == nil {
			hostport = u.Host
			if !strings.Contains(hostport, ":") {
				if u.Scheme == "http" {
					hostport = net.JoinHostPort(hostport, "4317")
				} else {
					hostport = net.JoinHostPort(hostport, "443")
				}
			}
		}
	}
	if hostport == "" {
		hostport = "localhost:4317"
	}

	insecure := envBool("OTEL_EXPORTER_OTLP_TRACES_INSECURE", "OTEL_EXPORTER_OTLP_INSECURE")
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(hostport),
		otlptracegrpc.WithHeaders(hdr),
		otlptracegrpc.WithTimeout(10 * time.Second),
	}
	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(&tls.Config{})))
	}
	return otlptracegrpc.New(ctx, opts...)
}

func protocol() string {
	p := strings.ToLower(strings.TrimSpace(firstNonEmpty("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "OTEL_EXPORTER_OTLP_PROTOCOL")))
	if p == "" {
		return "http/protobuf"
	}
	return p
}

func initResource(service, version string) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(service),
		semconv.ServiceVersionKey.String(version),
	}
	if env := strings.TrimSpace(os.Getenv("DEPLOY_ENV")); env != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentKey.String(env))
	}
	return resource.New(context.Background(),
		resource.WithFromEnv(), // picks up OTEL_RESOURCE_ATTRIBUTES if present
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithAttributes(attrs...),
	)
}

func InitTracerProvider(ctx context.Context, service, version string) (func(context.Context) error, error) {
	hdr := parseHeaders()
	proto := protocol()
	var (
		exp *otlptrace.Exporter
		err error
	)
	switch proto {
	case "http", "http/protobuf", "http_proto", "http+protobuf":
		exp, err = buildHTTPExporter(ctx, hdr)
	default:
		// treat anything else as gRPC
		exp, err = buildGRPCExporter(ctx, hdr)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to init OTLP exporter (%s): %w", proto, err)
	}

	res, err := initResource(service, version)
	if err != nil {
		return nil, fmt.Errorf("failed to init resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	// Friendly log line (keep this format—you were grepping for it)
	target := firstNonEmpty("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT")
	if target == "" {
		if proto == "http" || proto == "http/protobuf" {
			target = "https://otlp.arize.com/v1/traces"
		} else {
			target = "localhost:4317"
		}
	}
	// redacted headers
	red := map[string]string{}
	for k := range hdr {
		red[k] = "****"
	}
	log.Printf("OTEL exporter ready — protocol=%s, endpoint=%s, headers=%v\n", proto, target, red)

	return tp.Shutdown, nil
}

// Many servers already call an unexported variant—provide both.
func initTracerProvider(service, version string) (func(context.Context) error, error) {
	return InitTracerProvider(context.Background(), service, version)
}
