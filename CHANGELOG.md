# Changelog

## v0.6.1

### Removed

- `ErrNotStruct`; it was deprecated and has not been returned since v0.2.0.

## v0.6.0

### Added

- `binding` subpackage (`github.com/hushink/form/binding`) for decoding `*http.Request` values from path parameters, URL-encoded or multipart form bodies, JSON/XML bodies, query strings, headers, and cookies.
- Multi-source struct tags: one field may carry tags for several sources, and `binding.WithTagPriority` controls precedence (the first configured source has the highest priority).
- `binding.WithMultipartMaxMemory` for configuring `http.Request.ParseMultipartForm`.
- `binding.WithMaxBodyBytes` for limiting JSON/XML body reads, with `binding.ErrBodyTooLarge`.
- JSON and XML body sources use `encoding/json` / `encoding/xml`, apply only when the request Content-Type matches, and restore the body for downstream handlers.
- Path parameter binding via `http.Request.PathValue`; empty path values are treated as absent.
- `WithoutDefaults`, `WithoutRequiredChecks`, and `WithRequiredFallback` decoder options for composing multi-pass decoding.
- `ModeExplicitEmbedded`: like `ModeExplicit`, but anonymous embedded structs are traversed even without a tag, so their tagged fields are processed.
- `WithNamingStrategy` and the `NamingStrategy` type, with `SnakeCase`, `KebabCase`, and `LowerCamelCase` helpers. Strategies apply to fields without an explicit tag name; explicit tag names always win. The default remains the unchanged Go field name.
- `WithCaseInsensitive` for opt-in case-insensitive decoding. Exact matches win, with a deterministic lexicographic fallback; bracket map keys and indexes remain case-sensitive and encoding is unaffected.
- `binding.WithCaseInsensitive` for case-insensitive matching of map-style binding sources.
- `output` subpackage (`github.com/hushink/form/output`) for struct-to-output conversion: `Form`, `Query`, `QueryString`, `Header`, `JSON`, and `XML`, plus `WriteForm`, `WriteJSON`, `WriteXML`, and `SetHeader` HTTP helpers.
- Duplicate tag detection now runs after naming-strategy conversion, so strategy-generated collisions are reported by `WithRejectDuplicateTags(true)`.

### Changed

- Default naming and case behavior is unchanged: field names are used as written and map keys remain case-sensitive. Naming strategies and case-insensitive matching are opt-in.
- Map-style binding sources (`param`, `form`, `query`, `header`, `cookie`) bind only fields carrying a tag for that source; untagged fields are ignored. JSON and XML use standard `encoding/json` / `encoding/xml` matching.
- The default binding priority is now `param`, `form`, `json`, `xml`, `query`, `header`, `cookie`.
- Binding applies defaults before source values, so any actual value (including an explicit empty value) overrides a default.
- An empty value from a higher-priority binding source overrides a lower-priority value, including for numeric, time, and `[]byte` fields.
- Decoding into an existing non-nil pointer now updates the pointed-to value in place instead of replacing it, preserving fields absent from the current input. This is required for correct multi-pass binding.
- JSON/XML `default` and `required` options are not parsed; put those options on a map-style tag such as `form`.

## v0.5.0

### Added

- `WithRejectDuplicateTags(true)` and `ErrDuplicateTag` for reporting duplicate tag names.
- `WithAppendSlices(true)` for appending decoded slice values to existing slices.
- `WithRejectValueReceiverUnmarshalers(true)` for reporting value-receiver `UnmarshalValues`/`UnmarshalText` implementations that cannot mutate the destination.
- `WithTopLevelMarshaler(true)` for honoring a `Marshaler` implemented by the top-level struct.
- Support for decoding into non-nil unexported embedded pointer structs; a nil unexported embedded pointer targeted by input now reports a clear error instead of being silently ignored.
- `time.Duration` support: encoded as a duration string (`1m30s`) by default, with `WithDurationFormat(DurationNanoseconds)` to emit numeric nanoseconds. Decoding accepts both duration strings and numeric nanoseconds.

### Changed

- The default behavior remains unchanged: duplicate tags are allowed, slices are replaced, value-receiver unmarshalers are silent no-ops, and top-level structs are traversed by field.

## v0.4.0

### Added

- `AnonymousSeparate` mode and `WithAnonymousMode`.
- Configurable namespace prefix/suffix via `WithNamespacePrefix` / `WithNamespaceSuffix` (the suffix must not contain the prefix).
- Strict unknown-key checking via `WithStrictUnknownKeys(true)` with `UnknownKeyError`.
- `required` tag option and `EmptyFieldError`; `required` cannot be combined with `default`.
- `ZeroEmpty` decoder mode via `WithZeroEmpty(true)`.
- `encoding.TextMarshaler` / `encoding.TextUnmarshaler` fallback, including nil-pointer allocation and `net.IP`-style named byte slices.
- Additional tests for namespace separators, strict keys, required fields, zero-empty handling, and text interfaces.

### Changed

- `Marshaler` / `Unmarshaler` take precedence over Text interfaces; built-in `time.Time` handling takes precedence over `TextUnmarshaler`.
- `Encode` / `Decode` reject namespace configurations where the suffix contains the prefix.

## v0.3.0

### Added

- Interface field decoding: a nil interface receives the first value as a string; a non-nil interface is decoded through a fresh value of its dynamic type.
- Top-level non-struct encoding and decoding for scalars, slices, arrays, maps, and interfaces:
  - root scalar/slice values use the empty key `""`
  - top-level map entries use `[key]`
  - top-level slice/array elements use `[i]`
- `Mode` with `ModeImplicit` (default) and `ModeExplicit`, configurable via `WithMode`.

### Changed

- `ErrNotStruct` is no longer returned; it is kept as a deprecated sentinel for source compatibility.
- The path parser accepts the empty root key and leading bracket paths such as `[0]` and `[key]`.

## v0.2.0

### Added

- Nested struct support with dotted paths (`Address.City`).
- Indexed slice/array support (`Items[0].Name`) and map support (`Filter[key]`, `NestedMap[key][key]`).
- Flattening of anonymous embedded structs, including unexported embedded value structs.
- Aggregated errors: `DecodeErrors` and `EncodeErrors` keyed by field path, with sorted `Error()` and `Unwrap() []error`.
- Struct metadata caching for reusable `Encoder` / `Decoder` instances.
- `WithTagName`, `WithMaxArraySize`, and `WithTimeLayout` options.
- Maximum array index guard (default 10000) and maximum path/encoding depth guards.
- Transparent support for generic (instantiated) types and generic type aliases.
- Examples, benchmarks, fuzz tests, and a GitHub Actions CI workflow.

### Changed

- Decoding now follows go-playground/form value semantics:
  - Multiple values for a scalar use the first value.
  - Empty numeric/time/`[]byte` values are ignored instead of causing errors.
  - `bool` accepts `on/off/yes/no/ok`, and an empty value means `false`.
  - Empty slices are omitted during encoding.
  - Zero `time.Time` values are encoded unless `omitempty` is set.
- Custom `Marshaler` / `Unmarshaler` values in slices use indexed paths.
- `WithEncoderTagName` and `WithDecoderTagName` were replaced by `WithTagName`.
- The module path is now `github.com/hushink/form`; the package is `form`.
- Minimum Go version is 1.24.

### Removed

- Package-level `Encode` / `Decode` functions and their global cache. Use `Marshal` / `Unmarshal` or a reusable `Encoder` / `Decoder`.
- `EncoderOption` and `DecoderOption`; both constructors now accept `Option`.
