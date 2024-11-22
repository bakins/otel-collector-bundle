# OpenTelemetry Collector Velair Receiver

[OpenTelemetry Collector Receiver](https://opentelemetry.io/docs/collector/) for
[Velair VSD Marine Air Conditioners](https://uflex.it/velair/)


### Configuration

The following settings are required:
- `endpoint`: The base URL of the Velair air conditioner.

The following settings are optional:
- `collection_interval` (default = `10s`): This receiver collects metrics on an interval. This value must be a string readable by Golang's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration). Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.
- `initial_delay` (default = `1s`): defines how long this receiver waits before starting.

### Metrics

Currently:
- `velair.temperature`: ambient temperature