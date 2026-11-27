module github.com/krackl1n/careeria/services/vacancy

go 1.26.5

require (
	github.com/google/uuid v1.6.0
	github.com/krackl1n/careeria/clock v0.0.0-00010101000000-000000000000
	github.com/krackl1n/careeria/config v0.0.0
	github.com/krackl1n/careeria/grpc v0.0.0
	github.com/krackl1n/careeria/idgen v0.0.0-00010101000000-000000000000
	github.com/krackl1n/careeria/logger v0.0.0
	github.com/krackl1n/careeria/migration v0.0.0
	github.com/krackl1n/careeria/postgres v0.0.0
	github.com/krackl1n/careeria/runtime v0.0.0
	github.com/krackl1n/careeria/shutdown v0.0.0
	github.com/openfga/api/proto v0.0.0-20260723150800-6981fff8d33b
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260715232425-e75dac1f907d // indirect
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/krackl1n/careeria/authz v0.0.0
	github.com/krackl1n/careeria/contracts v0.0.0
	github.com/krackl1n/careeria/ctxmeta v0.0.0
	github.com/krackl1n/careeria/txmanager v0.0.0
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pressly/goose/v3 v3.27.2 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/grpc v1.82.1
	google.golang.org/protobuf v1.36.11
)

replace (
	github.com/krackl1n/careeria/config => ../../packages/go/config
	github.com/krackl1n/careeria/ctxmeta => ../../packages/go/ctxmeta
	github.com/krackl1n/careeria/grpc => ../../packages/go/grpc
	github.com/krackl1n/careeria/logger => ../../packages/go/logger
	github.com/krackl1n/careeria/migration => ../../packages/go/migration
	github.com/krackl1n/careeria/postgres => ../../packages/go/postgres
	github.com/krackl1n/careeria/runtime => ../../packages/go/runtime
	github.com/krackl1n/careeria/shutdown => ../../packages/go/shutdown
	github.com/krackl1n/careeria/txmanager => ../../packages/go/txmanager
)

replace github.com/krackl1n/careeria/contracts => ../../packages/go/contracts

replace github.com/krackl1n/careeria/clock => ../../packages/go/clock

replace github.com/krackl1n/careeria/idgen => ../../packages/go/idgen

replace github.com/krackl1n/careeria/authz => ../../packages/go/authz
