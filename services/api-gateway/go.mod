module github.com/krackl1n/careeria/services/api-gateway

go 1.26.5

require (
	github.com/go-chi/chi/v5 v5.3.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0
	github.com/krackl1n/careeria/closer v0.0.0
	github.com/krackl1n/careeria/config v0.0.0
	github.com/krackl1n/careeria/contracts v0.0.0
	github.com/krackl1n/careeria/ctxmeta v0.0.0
	github.com/krackl1n/careeria/grpc v0.0.0
	github.com/krackl1n/careeria/http v0.0.0
	github.com/krackl1n/careeria/jwt v0.0.0
	github.com/krackl1n/careeria/logger v0.0.0
	github.com/krackl1n/careeria/metrics v0.0.0
	github.com/krackl1n/careeria/runtime v0.0.0
	github.com/krackl1n/careeria/shutdown v0.0.0
	github.com/krackl1n/careeria/tracing v0.0.0
	github.com/prometheus/client_golang v1.23.2
	github.com/stretchr/testify v1.11.1
	github.com/swaggest/swgui v1.8.9
	golang.org/x/sync v0.22.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800
	google.golang.org/grpc v1.82.1
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-chi/cors v1.2.2 // indirect
	github.com/go-chi/httprate v0.16.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/vearutop/statigz v1.4.0 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.69.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260715232425-e75dac1f907d // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/krackl1n/careeria/http => ../../packages/go/http

replace github.com/krackl1n/careeria/jwt => ../../packages/go/jwt

replace github.com/krackl1n/careeria/ctxmeta => ../../packages/go/ctxmeta

replace github.com/krackl1n/careeria/config => ../../packages/go/config

replace github.com/krackl1n/careeria/contracts => ../../packages/go/contracts

replace github.com/krackl1n/careeria/grpc => ../../packages/go/grpc

replace github.com/krackl1n/careeria/closer => ../../packages/go/closer

replace github.com/krackl1n/careeria/logger => ../../packages/go/logger

replace github.com/krackl1n/careeria/metrics => ../../packages/go/metrics

replace github.com/krackl1n/careeria/runtime => ../../packages/go/runtime

replace github.com/krackl1n/careeria/shutdown => ../../packages/go/shutdown

replace github.com/krackl1n/careeria/tracing => ../../packages/go/tracing
