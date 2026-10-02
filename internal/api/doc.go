// Package api is the domain layer: typed models, operations and typed errors.
// Replace the sample Item with your own domain (HTTP API, database or files).
//
// Rules for this layer (enforced by .golangci.yml depguard and internal/archtest):
//   - no cobra, pflag, IOStreams or pkg/cmd imports, and no printing: return data;
//   - every operation that does I/O takes ctx first;
//   - no ambient state: env vars, config and the clock come in as fields or params;
//   - failures are typed errors (NotFoundError) that commands translate into
//     user-facing messages.
//
// See references/layers.md in the quality-cli skill.
package api
