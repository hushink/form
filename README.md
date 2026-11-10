# form

[中文文档](README_zh.md)

A Go library for bidirectional conversion between Go structs and `map[string][]string` (typically `url.Values`), designed for HTTP form data, query parameters, and similar use cases.

It follows the path and value semantics of [go-playground/form](https://github.com/go-playground/form), with a type-implemented `Marshaler`/`Unmarshaler` extension mechanism instead of type registration.

## Features

- Nested structs via dotted paths: `Address.City`
- Slices and arrays with indexes: `Items[0].Name`
- Maps with keys: `Filter[key]`, `NestedMap[key][key]`
- Anonymous embedded structs: flattened by default, or isolated with `AnonymousSeparate`
- Configurable namespace prefix/suffix for nested field keys
- All Go basic types, pointers, slices, arrays, maps, and nested structs
- Top-level non-struct targets: scalars, slices, arrays, maps, interfaces
- `interface{}` fields: a nil interface receives the first value as a string; a non-nil interface is decoded through its dynamic type
- `time.Time` / `*time.Time` support (RFC3339 by default, configurable)
- `time.Duration` support (encoded as `1m30s` by default; decoding accepts duration strings and numeric nanoseconds)
- `[]byte` / `*[]byte` treated as a single string
- Custom conversion via `Marshaler` / `Unmarshaler` and `encoding.TextMarshaler` / `encoding.TextUnmarshaler`
- Default values: `default=value`, multi-value `default=a|b|c`
- `required` and `omitempty` field options
- `ZeroEmpty` mode for resetting numeric/time/byte fields on empty input
- Strict unknown-key checking via `WithStrictUnknownKeys(true)`
- Optional duplicate-tag rejection, slice append mode, value-receiver diagnostics, and top-level `Marshaler` support
- `ModeImplicit` / `ModeExplicit` / `ModeExplicitEmbedded` field selection
- Aggregated errors (`DecodeErrors` / `EncodeErrors`) with `errors.Is` / `errors.As` support
- Struct metadata cache and configurable maximum array index
- Transparent support for generic (instantiated) types
- Configurable tag name via `WithTagName`
- Configurable naming strategies (`SnakeCase`, `KebabCase`, `LowerCamelCase`) for fields without an explicit tag name
- Opt-in case-insensitive decoding and binding for map-style sources; exact matches win
- `net/http` request binding via the `binding` subpackage: path parameters, form, JSON/XML bodies, query, headers, and cookies, with multiple source tags and configurable priority
- `output` subpackage for struct-to-form/query/header/JSON/XML output and HTTP response helpers

## Installation

```bash
go get github.com/hushink/form
```

Requires Go 1.24 or newer.

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/hushink/form"
)

type Address struct {
    City string `form:"city"`
    Zip  int    `form:"zip"`
}

type User struct {
    Name    string   `form:"name"`
    Tags    []string `form:"tags"`
    Address Address  `form:"address"`
}

func main() {
    user := User{
        Name: "Alice",
        Tags: []string{"admin", "user"},
        Address: Address{City: "Shanghai", Zip: 200000},
    }

    values, err := form.Marshal(user)
    if err != nil {
        panic(err)
    }
    fmt.Println(values["name"])         // [Alice]
    fmt.Println(values["tags"])         // [admin user]
    fmt.Println(values["address.city"]) // [Shanghai]

    var decoded User
    if err := form.Unmarshal(values, &decoded); err != nil {
        panic(err)
    }
    fmt.Printf("%+v\n", decoded)
}
```

`Marshal` returns a plain `map[string][]string`, which is assignable to `url.Values`.

## API

### Package-level functions

```go
func Marshal(v any) (map[string][]string, error)
func Unmarshal(data map[string][]string, v any) error
```

`Marshal` accepts any value (structs are encoded by field, other values use the root path). `Unmarshal` requires a non-nil pointer; the target may be a struct, map, slice, array, scalar, or interface.

### Encoder / Decoder

```go
func NewEncoder(opts ...Option) *Encoder
func (e *Encoder) Encode(v any) (map[string][]string, error)

func NewDecoder(opts ...Option) *Decoder
func (d *Decoder) Decode(data map[string][]string, v any) error
```

`Encoder` and `Decoder` cache struct metadata and are safe for concurrent use after construction. Reuse them when encoding or decoding the same types repeatedly.

### Options

```go
func WithTagName(name string) Option            // default: "form"
func WithMaxArraySize(size int) Option          // default: 10000
func WithTimeLayout(layout string) Option       // default: time.RFC3339
func WithMode(mode Mode) Option                 // default: ModeImplicit
func WithAnonymousMode(mode AnonymousMode) Option // default: AnonymousEmbed
func WithNamespacePrefix(prefix string) Option  // default: "."
func WithNamespaceSuffix(suffix string) Option  // default: ""
func WithStrictUnknownKeys(strict bool) Option  // default: false
func WithZeroEmpty(zero bool) Option            // default: false
func WithRejectDuplicateTags(reject bool) Option // default: false
func WithAppendSlices(appendSlices bool) Option // default: false
func WithRejectValueReceiverUnmarshalers(reject bool) Option // default: false
func WithTopLevelMarshaler(use bool) Option     // default: false
func WithDurationFormat(format DurationFormat) Option // default: DurationString
func WithNamingStrategy(strategy NamingStrategy) Option // default: nil (Go field name)
func WithCaseInsensitive(caseInsensitive bool) Option // default: false
func WithoutDefaults() Option                   // skip default= during decoding
func WithoutRequiredChecks() Option             // skip required validation during decoding
func WithRequiredFallback() Option              // a non-zero destination satisfies required
```

`WithMaxArraySize` limits indexed keys such as `Items[1000000]` to prevent oversized allocations from untrusted input.

`ModeImplicit` (the default) processes every exported field without a `-` tag. `ModeExplicit` processes only fields that carry the configured tag; a tag containing only options such as `form:",omitempty"` counts as tagged. `ModeExplicitEmbedded` also traverses anonymous embedded structs that have no tag, so their tagged fields are processed while other untagged fields remain skipped.

`AnonymousEmbed` (the default) promotes embedded fields into the parent namespace. `AnonymousSeparate` keeps them under the embedded type's own name (`Base.Field` instead of `Field`).

`WithNamespacePrefix` / `WithNamespaceSuffix` control nested key construction (`Parent<prefix>Child<suffix>`). The suffix must not contain the prefix; such a configuration is rejected by `Encode`/`Decode` because it cannot be parsed unambiguously.

`WithStrictUnknownKeys(true)` reports input keys that did not match any destination field as `UnknownKeyError` entries. `WithZeroEmpty(true)` resets numeric, time, and `[]byte` fields to their zero value when the input value is empty; pointer fields then point to the zero value.

`WithRejectDuplicateTags(true)` reports duplicate tag names within a struct as `ErrDuplicateTag` instead of letting both fields share a key. `WithAppendSlices(true)` appends decoded slice values to any existing slice instead of replacing it. `WithRejectValueReceiverUnmarshalers(true)` reports value-receiver `UnmarshalValues`/`UnmarshalText` implementations, which cannot mutate the field, as errors. `WithTopLevelMarshaler(true)` makes `Encode` honor a `Marshaler` implemented by the top-level struct instead of always traversing its fields.

`WithDurationFormat` controls `time.Duration` encoding: `DurationString` (default) emits values such as `1m30s`; `DurationNanoseconds` emits integer nanoseconds. Decoding always accepts both formats.

`WithoutDefaults`, `WithoutRequiredChecks`, and `WithRequiredFallback` are advanced options for composing several decoding passes. The `binding` subpackage uses them to give higher-priority request sources precedence without letting defaults or required checks from one pass interfere with another.

### Naming and case

```go
type NamingStrategy func(string) string

func SnakeCase(name string) string
func KebabCase(name string) string
func LowerCamelCase(name string) string

func WithNamingStrategy(strategy NamingStrategy) Option
func WithCaseInsensitive(caseInsensitive bool) Option
```

A naming strategy applies only when the tag does not provide an explicit name: an untagged field (in `ModeImplicit`) or a tag containing only options such as `form:",required"`. Explicit names such as `form:"custom_name"` always win. The default is nil, which keeps the Go field name unchanged.

```go
type User struct {
    UserID     int
    HTTPServer string
    Page       int `form:",default=1"`
}

enc := form.NewEncoder(form.WithNamingStrategy(form.SnakeCase))
// keys: user_id, http_server, page
```

`ModeExplicit` still skips untagged fields; the strategy only affects fields selected by an option-only tag. Duplicate detection via `WithRejectDuplicateTags(true)` runs after the strategy is applied, so `UserID` and `UserId` both mapping to `user_id` are reported as duplicates.

Case-sensitive matching is the default for map keys. `WithCaseInsensitive(true)` is opt-in and affects decoding only: an exact key match always wins, otherwise the lexicographically smallest case-insensitive match is used. Keys inside brackets (map keys and indexes) remain case-sensitive, and encoding is unaffected. The `binding` subpackage exposes the same option; headers are canonicalized, JSON uses `encoding/json`'s own case-insensitive matching, and XML remains case-sensitive.

### Interfaces

```go
type Marshaler interface {
    MarshalValues() ([]string, error)
}

type Unmarshaler interface {
    UnmarshalValues(values []string) error
}
```

Custom implementations take precedence over built-in conversion. Nil pointers are never passed to `MarshalValues`. `UnmarshalValues` receives all values for a struct field, and a one-element slice for each slice or map element.

Types implementing `encoding.TextMarshaler` / `encoding.TextUnmarshaler` are used after `Marshaler`/`Unmarshaler` and built-in `time.Time` handling, but before `[]byte`, basic-type, and struct conversion. This makes types such as `net.IP` work without adapters. `TextUnmarshaler` receives the first input value.

### Errors

Decoding and encoding continue after a field error. All errors are collected:

```go
type DecodeErrors map[string]error
type EncodeErrors map[string]error
```

The map key is the field path (for example `Address.City`, `Items[0].Name`, `Filter[key]`). `Error()` sorts keys for deterministic output, and both types implement `Unwrap() []error`, so `errors.Is` and `errors.As` work as expected.

Sentinel errors:

| Error | Meaning |
|-------|---------|
| `ErrInvalidValue` | Encoding input is nil |
| `ErrNilPointer` | Decode destination is a nil pointer |
| `ErrNotPointer` | Decode destination is not a pointer |

Additional decode error types:

| Error | Meaning |
|-------|---------|
| `UnknownKeyError` | Input key not matched by any field (strict mode) |
| `EmptyFieldError` | A field tagged `required` was missing or empty |
| `ErrDuplicateTag` | Duplicate tag names (with `WithRejectDuplicateTags(true)`) |

A failed `Decode` may have already modified some fields; a failed `Encode` returns the partially encoded map together with `EncodeErrors`.

## HTTP Binding

The `binding` subpackage binds `*http.Request` values to Go structs. A field may carry tags for several sources, and the source priority is configurable:

```go
import "github.com/hushink/form/binding"

type Search struct {
    ID      int    `param:"id" json:"id"`
    Keyword string `form:"keyword" query:"q"`
    Page    int    `form:"page" json:"page"`
    Token   string `header:"X-Token"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    var search Search
    if err := binding.Bind(r, &search); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    // ...
}
```

Supported sources:

| Source | Tag | Data |
|--------|-----|------|
| `SourceParam` | `param` | Path parameters via `http.Request.PathValue` |
| `SourceForm` | `form` | URL-encoded or multipart form body |
| `SourceJSON` | `json` | `application/json` or `+json` body |
| `SourceXML` | `xml` | `application/xml`, `text/xml`, or `+xml` body |
| `SourceQuery` | `query` | URL query string |
| `SourceHeader` | `header` | Request headers (canonical MIME names, e.g. `X-Token`) |
| `SourceCookie` | `cookie` | Request cookies |

The default priority is `param`, `form`, `json`, `xml`, `query`, `header`, `cookie`. The first tag in `WithTagPriority` has the highest priority: lower-priority values are bound first, then higher-priority values overwrite them. For example, `WithTagPriority(SourceQuery, SourceForm)` makes the query string win over the form body.

```go
func Bind(r *http.Request, v any, opts ...BindOption) error
func NewBinder(opts ...BindOption) *Binder
func (b *Binder) Bind(r *http.Request, v any) error

func WithTagPriority(tags ...string) BindOption
func WithMultipartMaxMemory(maxMemory int64) BindOption
func WithMaxBodyBytes(maxBytes int64) BindOption
func WithCaseInsensitive(caseInsensitive bool) BindOption
```

- Map-style sources bind only fields tagged for that source; untagged fields are ignored. For named nested structs, every parent field on the path must also carry a tag for that source. Anonymous embedded structs are traversed automatically, so their tagged fields bind without tagging the embedded field. JSON and XML follow standard `encoding/json` / `encoding/xml` matching, where tags are optional; use `json:"-"` / `xml:"-"` to exclude fields from those body sources.
- Binding names come from each source's tag, so naming style is explicit per source. `WithNamingStrategy` is an Encoder/Decoder option and is not applied to binding; JSON/XML keep their native naming rules.
- `WithCaseInsensitive(true)` makes map-style sources match tag names without regard to case; exact matches win. Path parameters are still fetched with `PathValue` using the tag name, so differently-cased router keys cannot match. JSON is already case-insensitive, and XML remains case-sensitive.
- `param` reads `http.Request.PathValue`, so the router must populate path values (the standard library does this for `ServeMux` patterns). Empty path values are treated as absent, and param tags should target flat scalar fields.
- JSON and XML sources apply only when the request `Content-Type` matches. The body is read fully, restored for downstream handlers, and decoded with `encoding/json` or `encoding/xml` at that source's position in the priority order. Empty bodies are skipped; malformed bodies stop binding with a wrapped error. The default read limit is 32 MiB and is configurable with `WithMaxBodyBytes`.
- Defaults are applied before source values, so any actual value (including an explicit empty value) overrides a default.
- When a field has tags for several sources, put `required` on the highest-priority tag. A non-zero value already bound from a lower-priority source or a default satisfies it. `default` and `required` are recognized on map-style tags (`param`, `form`, `query`, `header`, `cookie`); JSON/XML fields can add a form tag when those checks are needed.
- Decode errors from all sources are merged into `form.DecodeErrors`; request parsing errors stop binding and wrap the underlying error.
- Every configured source is decoded when applicable, so an error in a lower-priority source is still reported even when a higher-priority source supplies a valid replacement.
- `ErrNilRequest`, `ErrUnsupportedSource`, and `ErrBodyTooLarge` are sentinel errors; parsing errors wrap the underlying error.
- A `Binder` is safe for concurrent use and caches metadata; reuse it for repeated requests.
- `WithMultipartMaxMemory` controls `http.Request.ParseMultipartForm`; call `r.MultipartForm.RemoveAll()` after handling uploads if needed.

## HTTP Output

The `output` subpackage converts structs into HTTP-oriented output formats:

```go
import "github.com/hushink/form/output"

type Search struct {
    Query string `query:"q"`
    Page  int    `query:"page"`
}

query, err := output.QueryString(Search{Query: "go", Page: 2})
// page=2&q=go

values, err := output.Form(v)   // form tags -> url.Values
header, err := output.Header(v) // header tags -> http.Header (canonical names)
jsonData, err := output.JSON(v) // encoding/json
xmlData, err := output.XML(v)   // encoding/xml

output.WriteForm(w, v)
output.WriteJSON(w, v)
output.WriteXML(w, v)
output.SetHeader(w, v)
```

- `Form`, `Query`, and `Header` use the corresponding source tags and include only tagged fields by default (`ModeExplicitEmbedded`). Named nested parent fields need the source tag for nested fields to be included. Pass `form.WithMode(form.ModeImplicit)` or other form options to customize naming and selection.
- JSON and XML use standard `encoding/json` / `encoding/xml` tags and matching; form naming options do not apply.
- The writers set `Content-Type` only when the response does not already have one; status codes are left to the caller.
- Header names are canonicalized; header tags should target flat fields.

## Path Syntax

| Syntax | Meaning |
|--------|---------|
| `Name` | Top-level field |
| `Address.City` | Nested struct field |
| `Items[0].Name` | Slice/array element field |
| `Filter[key]` | Map entry |
| `NestedMap[key][key2]` | Nested map entry |
| `Matrix[0][1]` | Nested slice/array element |
| `MapValue[k][0]` | Slice stored as a map value |
| `""` (empty key) | Root value for a top-level scalar or slice |
| `[0]` | Element of a top-level slice/array |
| `[key]` | Entry of a top-level map |

Rules:

- Values for a slice of basic values are sent as repeated values (`Tags=a&Tags=b`) when the slice is a direct struct field. Indexed keys (`Tags[0]=a`) are also accepted, and can be mixed with repeated values.
- Slices of pointers, structs, arrays, maps, and custom marshalers are encoded with indexes (`Items[0].Name`, `Ptrs[0]`, `Times[0]`).
- Slices nested inside another slice/array or a map value always use indexes (`Matrix[0][1]`, `MapValue[k][0]`), matching go-playground/form.
- Empty field segments (for example `.a` or `a..b`), empty brackets, and malformed brackets are reported as decode errors. The empty key `""` is reserved for the root value.
- Unknown keys are ignored by default; use `WithStrictUnknownKeys(true)` to report them.

Namespace separators are configurable. With `WithNamespacePrefix("_")`, `Address.City` becomes `Address_City`; with `WithNamespaceSuffix("_end")`, it becomes `Address.City_end`. The suffix is appended to every nested field name (not root fields) and must not contain the prefix.

### Top-level targets

Non-struct targets use the root path:

```go
form.Marshal(map[string]int{"a": 1}) // {"[a]": {"1"}}
form.Marshal([]string{"a", "b"})     // {"": {"a", "b"}}
form.Marshal(42)                     // {"": {"42"}}

var m map[string]int
form.Unmarshal(map[string][]string{"[a]": {"1"}}, &m) // m[a] = 1

var s []string
form.Unmarshal(map[string][]string{"": {"a", "b"}}, &s) // s = [a b]
```

## Struct Tags

```
`form:"field_name,option1,option2,..."`
```

| Option | Example | Meaning |
|--------|---------|---------|
| Name | `form:"custom_name"` | Map key prefix |
| Skip | `form:"-"` | Ignore the field |
| Omit empty | `form:"name,omitempty"` | Skip zero values when encoding |
| Required | `form:"name,required"` | Report `EmptyFieldError` when the input value is missing or empty |
| Default | `form:"name,default=value"` | Use the value when the key is missing |
| Multi-value default | `form:"name,default=a\|b\|c"` | `\|` separates default values |

Defaults are applied only when the key is missing (or present with an empty value list). An explicitly supplied `0` or `""` does not trigger the default. A field cannot combine `required` with `default`; decoding reports a configuration error for that combination.

With `omitempty` and a default, a field is omitted when its encoded values equal the default. The comparison is textual: `default=3.140` does not match the encoded value `3.14`.

## Value Semantics

Decoding:

- Missing keys leave fields untouched; defaults apply if configured.
- Decoding into a non-nil pointer updates the pointed-to value in place; fields absent from the input stay unchanged.
- A scalar field with multiple input values uses the first value.
- `string` receives `""` as an empty string; `*string` becomes a pointer to `""`.
- `bool` accepts `1/t/T/true/TRUE/True/on/yes/ok` and `0/f/F/false/FALSE/False/off/no/""`.
- For numeric, float, `time.Time`, and `[]byte` fields, an empty string is ignored (the field is not modified and no error is reported).
- A slice is built from all input values. Empty elements become zero values (`""`, `0`, `false`, zero time).
- `[]byte` is treated as a single string, not as a list of numbers.
- A nil interface receives the first value as a string. A non-nil interface is decoded by creating a fresh value of its dynamic type; nested keys require that concrete type to be a struct, slice, map, or array.
- `required` checks the input: an absent key, an empty value list, or values that are all empty fails with `EmptyFieldError`. An explicit `0` or `false` is a provided value.
- `WithZeroEmpty(true)` turns an empty input value into the field's zero value for numbers, time, and `[]byte`; pointer fields then point to the zero value.
- `encoding.TextUnmarshaler` receives the first input value (including an empty string); `Marshaler`/`Unmarshaler` take precedence over Text interfaces.
- `time.Duration` accepts values such as `1s`, `1m30s`, `500ms`, or numeric nanoseconds (`1500000000`).

Encoding:

- Nil pointers, nil interfaces, nil/empty slices, nil/empty maps, and empty arrays produce no keys.
- `[]byte` is encoded as one string value.
- `time.Time` is encoded with the configured layout; without `omitempty`, a zero time is encoded too.
- `omitempty` skips nil and zero values. Empty slices/maps produce no output anyway.
- Anonymous embedded structs are flattened into the parent's keys.

## Supported Types

| Category | Types |
|----------|-------|
| Basic | `string`, `bool`, `int*`, `uint*`, `float32`, `float64` |
| Pointers | Pointers to any supported type (including nested pointers) |
| Containers | Slices, arrays, maps, nested structs |
| Time | `time.Time`, `*time.Time`, `time.Duration`, `*time.Duration` |
| Bytes | `[]byte` and named types whose element type is exactly `byte` |
| Interface | `interface{}` and other interface fields |
| Custom | Types implementing `Marshaler` / `Unmarshaler` or `encoding.TextMarshaler` / `TextUnmarshaler` |

Interface fields can be encoded (the dynamic value is used) and decoded (nil interfaces become strings; non-nil interfaces decode through their dynamic type). Function, channel, complex, and unsafe pointer fields are unsupported.

## Generics

Generic types work transparently; reflection sees each instantiated type (`Box[int]`, `Box[string]`) as a distinct type, and the metadata cache is keyed by that type.

```go
type Box[T any] struct {
    Value T `form:"value"`
}

values, _ := form.Marshal(Box[int]{Value: 42}) // value=[42]

var box Box[string]
_ = form.Unmarshal(map[string][]string{"value": {"hello"}}, &box)
```

Generic aliases are supported (Go 1.24+):

```go
type IntBox = Box[int]
```

## Limitations

- Struct tag names must not contain the active namespace prefix, the namespace suffix, `[` or `]`; they are path separators. With the default options this means `.`, `[` and `]`.
- Map keys must be non-empty and must not contain `[` or `]`.
- Default values cannot contain `|` (the multi-value separator) or `,` (the tag option separator).
- Defaults are supported for scalar, pointer, time, `[]byte`, array, and slice-of-scalar fields; defaults on map and struct fields are not applied.
- The namespace suffix must not contain the namespace prefix; `Encode`/`Decode` reject such configurations.
- A field cannot combine `required` with `default`; decoding reports a configuration error.
- A nil interface can only receive a string; decoding nested keys into an interface requires a non-nil dynamic value of a concrete struct/slice/map/array type.
- Unexported embedded value structs and non-nil unexported embedded pointer structs are supported. A nil unexported embedded pointer cannot be allocated through reflection; input targeting it reports an error.
- Duplicate tag names are allowed by default (both fields share the key); enable `WithRejectDuplicateTags(true)` to report `ErrDuplicateTag`.
- Decoding replaces slice values by default; enable `WithAppendSlices(true)` to append to existing slices.
- Value-receiver `UnmarshalValues`/`UnmarshalText` implementations cannot mutate the field and are silent no-ops by default; enable `WithRejectValueReceiverUnmarshalers(true)` to report them.
- Top-level structs are encoded by field by default, even if they implement `Marshaler`; enable `WithTopLevelMarshaler(true)` to use the interface.
- Array index overflow and invalid slice indexes are reported as decode errors instead of being ignored.
- Defaults are not applied inside pointer structs that receive no input at all.
- Encoding a cyclic pointer graph reports a maximum-depth error instead of panicking.

## Continuous integration

The repository includes `.github/workflows/ci.yml`, which runs `go vet`, `go test -race`, fuzz smoke tests, and golangci-lint on Go 1.24–1.27.

- **GitHub**: push or mirror the repository to GitHub to run the workflow automatically.
- **Gitee Go**: Gitee does not execute GitHub Actions workflows. A pipeline template is provided in `.workflow/ci.yml`; enable Gitee Go in the repository settings (adjust `golangVersion` if Go 1.24 is not available in your Gitee Go environment). The pipeline runs the same commands:
  ```bash
  go vet ./...
  go test -race -coverprofile=coverage.out ./...
  go test -run '^$' -fuzz=FuzzParseKeyPath -fuzztime=10s .
  go test -run '^$' -fuzz=FuzzDecode -fuzztime=10s .
  golangci-lint run ./...
  ```
- **Local**: `make ci` runs fmt-check, vet, lint, race tests, and fuzz smoke tests. The race detector requires cgo/gcc.

## License

Apache-2.0
