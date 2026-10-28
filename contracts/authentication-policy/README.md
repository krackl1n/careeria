# Authentication policy

Portable protobuf method option and generator for per-RPC authentication policy.

`proto/authentication/policy/v1/policy.proto` defines `PUBLIC` and `REQUIRED`.
`cmd/protoc-gen-authentication-policy` checks every generated RPC and emits a
Go map indexed by canonical gRPC method name. The generator itself has no
dependency on careeria; the consuming project supplies the Go package for the
option proto through Buf managed mode.

Add `proto` as a module in the consuming Buf workspace, import
`authentication/policy/v1/policy.proto` in service files, and set the option
on each RPC:

```proto
option (authentication.policy.v1.authentication) = AUTHENTICATION_POLICY_REQUIRED;
```

The schema intentionally has no `go_package`. The consuming project's Buf
managed mode assigns its own Go import path for
`authentication/policy/v1/policy.proto`. Generate that file with `protoc-gen-go`
and generate the policy map from the API's proto module with this plugin. Use
`strategy: all` so the plugin sees all RPCs in one request. This repository's
`contracts/buf.gen.policy.yaml` and `contracts/buf.gen.yaml` are working examples.

The plugin processes every generated RPC and rejects missing or unspecified
policies. A project can pass `exclude_package_segment=admin` to omit an
internal namespace from the public policy map. The output is placed beside
the generated `policy.pb.go` file and uses its Go package. The option proto
and generator can be copied together; adjust `go.mod` to the destination
module path if the copied tool will be imported by Go package name. If the
consumer uses `go.work`, add this Go module to it (as this repository does),
or run the generator with `GOWORK=off`.

The generated map describes authentication only. JWT validation and access
checks are performed by the consuming runtime or generated infrastructure
configuration.

Implementation is split into the thin `cmd/protoc-gen-authentication-policy`
entry point and `internal/generator`: descriptor discovery, policy decoding,
RPC collection and Go rendering live in separate files. The generator's tests
construct a complete `CodeGeneratorRequest` and check deterministic output,
package exclusion and failures for missing, unknown, duplicated and malformed
policy values. Run them independently with:

```sh
GOWORK=off go test ./...
```

In this repository, `make proto-generate` exercises the real Buf integration;
`packages/go/contracts/authentication_test.go` checks the resulting Go map
against every public service descriptor.
