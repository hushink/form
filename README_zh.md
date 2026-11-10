# form

[English](README.md)

一个 Go 语言库，用于 Go 结构体与 `map[string][]string`（通常是 `url.Values`）之间的双向转换，适用于 HTTP 表单、查询参数等场景。

路径与取值语义参考 [go-playground/form](https://github.com/go-playground/form)，但自定义类型采用类型实现 `Marshaler` / `Unmarshaler` 的方式，无需注册。

## 特性

- 嵌套结构体：`Address.City`
- 带索引的切片/数组：`Items[0].Name`
- map：`Filter[key]`、`NestedMap[key][key]`
- 匿名内嵌结构体：默认扁平化，也可用 `AnonymousSeparate` 独立命名空间
- 可配置嵌套 key 的 namespace 前缀/后缀
- 支持全部 Go 基础类型、指针、切片、数组、map、嵌套结构体
- 顶层非结构体目标：标量、切片、数组、map、interface
- `interface{}` 字段：nil interface 取首值作为字符串；非 nil interface 通过动态类型解码
- `time.Time` / `*time.Time`（默认 RFC3339，可配置）
- `time.Duration` 支持（默认编码为 `1m30s` 这类字符串；解码兼容字符串和数字纳秒）
- `[]byte` / `*[]byte` 按单个字符串处理
- 通过 `Marshaler` / `Unmarshaler` 以及 `encoding.TextMarshaler` / `TextUnmarshaler` 自定义转换
- 默认值：`default=value`，多值 `default=a|b|c`
- `required` 与 `omitempty`
- `ZeroEmpty` 模式：空输入将数字/时间/字节字段重置为零值
- `WithStrictUnknownKeys(true)` 严格未知 key 检查
- 可选的重复 tag 拒绝、切片追加模式、value receiver 诊断与顶层 Marshaler 支持
- `ModeImplicit` / `ModeExplicit` / `ModeExplicitEmbedded` 字段选择
- 错误聚合（`DecodeErrors` / `EncodeErrors`），支持 `errors.Is` / `errors.As`
- 结构体元数据缓存和可配置的最大数组索引
- 透明支持泛型实例化类型
- 可通过 `WithTagName` 自定义 tag 名称
- 未显式命名时可使用 `SnakeCase`、`KebabCase`、`LowerCamelCase` 命名策略
- 可选的大小写不敏感解码与绑定（map 式来源；精确匹配优先）
- `binding` 子包提供 `net/http` 请求绑定：路径参数、表单、JSON/XML body、查询参数、请求头与 Cookie，支持多来源标签与可配置优先级
- `output` 子包提供结构体到 form/query/header/JSON/XML 的输出与 HTTP 响应辅助函数

## 安装

```bash
go get github.com/hushink/form
```

需要 Go 1.24 或更高版本。

## 快速开始

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

`Marshal` 返回普通的 `map[string][]string`，可直接赋值给 `url.Values`。

## API

### 包级函数

```go
func Marshal(v any) (map[string][]string, error)
func Unmarshal(data map[string][]string, v any) error
```

`Marshal` 接受任意值（结构体按字段编码，其他值使用根路径）；`Unmarshal` 要求非 nil 指针，目标可以是结构体、map、切片、数组、标量或 interface。

### Encoder / Decoder

```go
func NewEncoder(opts ...Option) *Encoder
func (e *Encoder) Encode(v any) (map[string][]string, error)

func NewDecoder(opts ...Option) *Decoder
func (d *Decoder) Decode(data map[string][]string, v any) error
```

`Encoder` / `Decoder` 会缓存结构体元数据，构造完成后可安全并发使用。重复编解码同一批类型时建议复用实例。

### 选项

```go
func WithTagName(name string) Option              // 默认 "form"
func WithMaxArraySize(size int) Option            // 默认 10000
func WithTimeLayout(layout string) Option         // 默认 time.RFC3339
func WithMode(mode Mode) Option                   // 默认 ModeImplicit
func WithAnonymousMode(mode AnonymousMode) Option // 默认 AnonymousEmbed
func WithNamespacePrefix(prefix string) Option    // 默认 "."
func WithNamespaceSuffix(suffix string) Option    // 默认 ""
func WithStrictUnknownKeys(strict bool) Option    // 默认 false
func WithZeroEmpty(zero bool) Option              // 默认 false
func WithRejectDuplicateTags(reject bool) Option  // 默认 false
func WithAppendSlices(appendSlices bool) Option   // 默认 false
func WithRejectValueReceiverUnmarshalers(reject bool) Option // 默认 false
func WithTopLevelMarshaler(use bool) Option       // 默认 false
func WithDurationFormat(format DurationFormat) Option // 默认 DurationString
func WithNamingStrategy(strategy NamingStrategy) Option // 默认 nil（使用 Go 字段名）
func WithCaseInsensitive(caseInsensitive bool) Option // 默认 false
func WithoutDefaults() Option                     // 解码时跳过 default=
func WithoutRequiredChecks() Option               // 解码时跳过 required 校验
func WithRequiredFallback() Option                // 非零目标值可满足 required
```

`WithMaxArraySize` 限制 `Items[1000000]` 这类索引键，防止不可信输入导致超大分配。

`ModeImplicit`（默认）处理所有不带 `-` tag 的导出字段；`ModeExplicit` 只处理带配置 tag 的字段（`form:",omitempty"` 这类只含选项的 tag 也算带 tag）。`ModeExplicitEmbedded` 还会遍历没有 tag 的匿名内嵌结构体，处理其中带 tag 的字段，其他未加 tag 的字段仍然跳过。

`AnonymousEmbed`（默认）将内嵌字段提升到父级 namespace；`AnonymousSeparate` 保留在嵌入类型自己的名称下（`Base.Field` 而不是 `Field`）。

`WithNamespacePrefix` / `WithNamespaceSuffix` 控制嵌套 key 的构造（`Parent<prefix>Child<suffix>`）。后缀不能包含前缀；这种配置无法无歧义解析，`Encode`/`Decode` 会直接返回错误。

`WithStrictUnknownKeys(true)` 将没有匹配到任何字段的输入 key 作为 `UnknownKeyError` 返回。`WithZeroEmpty(true)` 在输入为空时把数字、时间和 `[]byte` 字段重置为零值；指针字段会指向零值。

`WithRejectDuplicateTags(true)` 把同一 struct 内的重复 tag 名作为 `ErrDuplicateTag` 返回。`WithAppendSlices(true)` 将解码结果追加到已有切片而不是替换。`WithRejectValueReceiverUnmarshalers(true)` 把无法修改字段的 value receiver `UnmarshalValues`/`UnmarshalText` 作为错误返回。`WithTopLevelMarshaler(true)` 让 `Encode` 优先使用顶层 struct 的 `Marshaler`，而不是按字段遍历。

`WithDurationFormat` 控制 `time.Duration` 编码：`DurationString`（默认）输出 `1m30s` 这类字符串；`DurationNanoseconds` 输出整数纳秒。解码始终同时接受两种格式。

`WithoutDefaults`、`WithoutRequiredChecks` 与 `WithRequiredFallback` 是用于组合多趟解码的高级选项。`binding` 子包使用它们让高优先级请求来源覆盖低优先级值，同时避免某一趟的默认值或 required 校验干扰其他来源。

### 命名与大小写

```go
type NamingStrategy func(string) string

func SnakeCase(name string) string
func KebabCase(name string) string
func LowerCamelCase(name string) string

func WithNamingStrategy(strategy NamingStrategy) Option
func WithCaseInsensitive(caseInsensitive bool) Option
```

命名策略只在 tag 没有提供显式名称时生效：未加 tag 的字段（`ModeImplicit`）或只含选项的 tag（例如 `form:",required"`）。显式名称（例如 `form:"custom_name"`）始终优先。默认值为 nil，即保持 Go 字段名不变。

```go
type User struct {
    UserID     int
    HTTPServer string
    Page       int `form:",default=1"`
}

enc := form.NewEncoder(form.WithNamingStrategy(form.SnakeCase))
// key：user_id、http_server、page
```

`ModeExplicit` 仍会跳过未加 tag 的字段，命名策略只影响由“仅选项 tag”选中的字段。`WithRejectDuplicateTags(true)` 在应用命名策略后检测重复名，因此 `UserID` 与 `UserId` 都转换为 `user_id` 时会返回重复错误。

map key 默认区分大小写。`WithCaseInsensitive(true)` 是需要显式开启的解码选项：精确匹配始终优先，否则使用字典序最小的不区分大小写匹配；方括号内的 map key/索引仍区分大小写，编码不受影响。`binding` 子包提供同名选项；header 已规范化，JSON 使用 `encoding/json` 自身的大小写不敏感匹配，XML 仍区分大小写。

### 接口

```go
type Marshaler interface {
    MarshalValues() ([]string, error)
}

type Unmarshaler interface {
    UnmarshalValues(values []string) error
}
```

自定义实现优先于内置转换；nil 指针不会被传给 `MarshalValues`。`UnmarshalValues` 对结构体字段接收全部 value，对每个切片/map 元素接收单元素切片。

实现 `encoding.TextMarshaler` / `encoding.TextUnmarshaler` 的类型在 `Marshaler`/`Unmarshaler` 和内置 `time.Time` 处理之后、`[]byte`/基础类型/结构体转换之前生效；`net.IP` 这类类型因此无需适配器。`TextUnmarshaler` 接收首个输入 value。

### 错误

解码和编码遇到字段错误后会继续处理，所有错误集中返回：

```go
type DecodeErrors map[string]error
type EncodeErrors map[string]error
```

map 的 key 是字段路径（例如 `Address.City`、`Items[0].Name`、`Filter[key]`）。`Error()` 对 key 排序以保证输出稳定；两个类型都实现 `Unwrap() []error`，因此 `errors.Is` / `errors.As` 可用。

哨兵错误：

| 错误 | 含义 |
|------|------|
| `ErrInvalidValue` | 编码输入为 nil |
| `ErrNilPointer` | 解码目标是 nil 指针 |
| `ErrNotPointer` | 解码目标不是指针 |

额外的解码错误类型：

| 错误 | 含义 |
|------|------|
| `UnknownKeyError` | 严格模式下没有匹配到字段的输入 key |
| `EmptyFieldError` | `required` 字段缺失或为空 |
| `ErrDuplicateTag` | 重复 tag 名（配合 `WithRejectDuplicateTags(true)`） |

解码失败时目标结构体可能已被部分修改；编码失败时会同时返回部分编码结果和 `EncodeErrors`。

## HTTP 绑定

`binding` 子包把 `*http.Request` 绑定到 Go 结构体。同一个字段可以携带多个来源的 tag，并且来源优先级可配置：

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

支持的来源：

| 来源 | Tag | 数据 |
|------|-----|------|
| `SourceParam` | `param` | 通过 `http.Request.PathValue` 读取的路径参数 |
| `SourceForm` | `form` | URL-encoded 或 multipart 表单 body |
| `SourceJSON` | `json` | `application/json` 或 `+json` body |
| `SourceXML` | `xml` | `application/xml`、`text/xml` 或 `+xml` body |
| `SourceQuery` | `query` | URL 查询字符串 |
| `SourceHeader` | `header` | 请求头（使用规范的 MIME 名称，例如 `X-Token`） |
| `SourceCookie` | `cookie` | 请求 Cookie |

默认优先级为 `param`、`form`、`json`、`xml`、`query`、`header`、`cookie`。`WithTagPriority` 中排在前面的 tag 优先级更高：先绑定低优先级来源，再由高优先级来源覆盖。例如 `WithTagPriority(SourceQuery, SourceForm)` 会让查询参数优先于表单 body。

```go
func Bind(r *http.Request, v any, opts ...BindOption) error
func NewBinder(opts ...BindOption) *Binder
func (b *Binder) Bind(r *http.Request, v any) error

func WithTagPriority(tags ...string) BindOption
func WithMultipartMaxMemory(maxMemory int64) BindOption
func WithMaxBodyBytes(maxBytes int64) BindOption
func WithCaseInsensitive(caseInsensitive bool) BindOption
```

- 地图式来源只绑定携带该来源 tag 的字段；没有 tag 的字段会被忽略。命名嵌套结构体路径上的每一级父字段也必须携带该来源的 tag；匿名内嵌结构体会自动遍历，其带 tag 的字段无需给内嵌字段本身加 tag。JSON 与 XML 遵循标准 `encoding/json` / `encoding/xml` 匹配规则，tag 是可选的；可用 `json:"-"` / `xml:"-"` 排除字段。
- binding 的名称来自各来源 tag，因此命名风格按来源显式指定。`WithNamingStrategy` 是 Encoder/Decoder 选项，不作用于 binding；JSON/XML 保持各自的命名规则。
- `WithCaseInsensitive(true)` 让 map 式来源忽略 tag 名称大小写；精确匹配优先。param 仍通过 `PathValue(tagName)` 读取，因此路由中大小写不同的 key 无法匹配。JSON 本身大小写不敏感，XML 保持区分大小写。
- `param` 通过 `http.Request.PathValue` 读取，因此路由必须填充路径参数（标准库 `ServeMux` 模式会自动填充）。空路径值视为缺失；param tag 应作用于扁平标量字段。
- JSON 与 XML 来源仅在请求 `Content-Type` 匹配时生效。body 会被完整读取、恢复给后续 handler，并按其优先级位置使用 `encoding/json` 或 `encoding/xml` 解码。空 body 会被跳过；格式错误的 body 会停止绑定并返回包装后的错误。默认读取上限为 32 MiB，可通过 `WithMaxBodyBytes` 调整。
- 默认值在来源值之前应用，因此任何实际值（包括显式空值）都会覆盖默认值。
- 字段有多个来源 tag 时，把 `required` 放在优先级最高的 tag 上；低优先级来源或默认值已经绑定的非零值也可满足它。`default` 与 `required` 仅在地图式来源 tag（`param`、`form`、`query`、`header`、`cookie`）上识别；JSON/XML 字段需要这些校验时可以额外添加 form tag。
- 所有来源的解码错误会合并为 `form.DecodeErrors`；请求解析错误会停止绑定并包装底层错误。
- 所有适用来源都会参与解码，因此即使高优先级来源提供了有效值，低优先级来源的错误仍会被报告。
- `ErrNilRequest`、`ErrUnsupportedSource` 与 `ErrBodyTooLarge` 是哨兵错误；解析错误会包装底层错误。
- `Binder` 可安全并发使用并缓存元数据；重复请求建议复用。
- `WithMultipartMaxMemory` 控制 `http.Request.ParseMultipartForm`；处理上传后如需清理临时文件，请调用 `r.MultipartForm.RemoveAll()`。

## HTTP 输出

`output` 子包把结构体转换为 HTTP 输出格式：

```go
import "github.com/hushink/form/output"

type Search struct {
    Query string `query:"q"`
    Page  int    `query:"page"`
}

query, err := output.QueryString(Search{Query: "go", Page: 2})
// page=2&q=go

values, err := output.Form(v)   // form tag -> url.Values
header, err := output.Header(v) // header tag -> http.Header（规范名称）
jsonData, err := output.JSON(v) // encoding/json
xmlData, err := output.XML(v)   // encoding/xml

output.WriteForm(w, v)
output.WriteJSON(w, v)
output.WriteXML(w, v)
output.SetHeader(w, v)
```

- `Form`、`Query`、`Header` 使用对应来源 tag，默认只输出带该 tag 的字段（`ModeExplicitEmbedded`）；命名嵌套结构体的父字段也需要携带该来源 tag。可传 `form.WithMode(form.ModeImplicit)` 或其他 form 选项调整字段选择与命名。
- JSON 与 XML 使用标准 `encoding/json` / `encoding/xml` tag 与匹配规则；form 的命名选项不适用。
- 写入函数仅在响应尚无 `Content-Type` 时设置；状态码由调用方决定。
- Header 名称会规范化；header tag 应作用于扁平字段。

## 路径语法

| 语法 | 含义 |
|------|------|
| `Name` | 顶层字段 |
| `Address.City` | 嵌套结构体字段 |
| `Items[0].Name` | 切片/数组元素字段 |
| `Filter[key]` | map 条目 |
| `NestedMap[key][key2]` | 嵌套 map 条目 |
| `Matrix[0][1]` | 嵌套切片/数组元素 |
| `MapValue[k][0]` | 作为 map value 的切片 |
| `""`（空 key） | 顶层标量/切片的根值 |
| `[0]` | 顶层切片/数组元素 |
| `[key]` | 顶层 map 条目 |

规则：

- 直接作为结构体字段的基础类型切片使用重复 value：`Tags=a&Tags=b`；也接受 `Tags[0]=a`，两种写法可以混用。
- 指针、结构体、数组、map、自定义类型作为切片元素时使用索引编码（`Items[0].Name`、`Ptrs[0]`、`Times[0]`）。
- 嵌套在另一个切片/数组或 map value 中的切片始终使用索引（`Matrix[0][1]`、`MapValue[k][0]`），与 go-playground/form 一致。
- 空字段段（例如 `.a` 或 `a..b`）、空方括号、格式错误的方括号会作为解码错误返回；空 key `""` 专用于根值。
- 默认忽略未知 key；使用 `WithStrictUnknownKeys(true)` 可报告它们。

namespace 分隔符可配置：`WithNamespacePrefix("_")` 会把 `Address.City` 变成 `Address_City`；`WithNamespaceSuffix("_end")` 会变成 `Address.City_end`。后缀会附加到每个嵌套字段名（根字段除外），且不能包含前缀。

### 顶层目标

非结构体目标使用根路径：

```go
form.Marshal(map[string]int{"a": 1}) // {"[a]": {"1"}}
form.Marshal([]string{"a", "b"})     // {"": {"a", "b"}}
form.Marshal(42)                     // {"": {"42"}}

var m map[string]int
form.Unmarshal(map[string][]string{"[a]": {"1"}}, &m) // m[a] = 1

var s []string
form.Unmarshal(map[string][]string{"": {"a", "b"}}, &s) // s = [a b]
```

## 结构体标签

```
`form:"field_name,option1,option2,..."`
```

| 选项 | 示例 | 说明 |
|------|------|------|
| 字段名 | `form:"custom_name"` | 自定义 map key 前缀 |
| 忽略字段 | `form:"-"` | 编码和解码时跳过 |
| 省略空值 | `form:"name,omitempty"` | 编码时跳过零值 |
| 必填 | `form:"name,required"` | 输入缺失或为空时返回 `EmptyFieldError` |
| 默认值 | `form:"name,default=value"` | key 缺失时使用 |
| 多值默认 | `form:"name,default=a\|b\|c"` | 使用 `\|` 分隔 |

默认值仅在 key 缺失（或 key 存在但 value 列表为空）时生效；显式传入 `0` 或 `""` 不会触发默认值。

`omitempty` 与 `default` 同时使用时，字段编码结果等于默认值才省略。比较是文本级的：`default=3.140` 不会匹配编码后的 `3.14`。

## 取值语义

解码：

- key 缺失时保持字段原值；有默认值则应用默认值。
- 解码到非 nil 指针时会原地更新指向值；输入中没有出现的字段保持原值。
- 标量字段收到多个值时取第一个。
- `string` 收到 `""` 时设为空串；`*string` 得到指向 `""` 的指针。
- `bool` 接受 `1/t/T/true/TRUE/True/on/yes/ok` 和 `0/f/F/false/FALSE/False/off/no/""`。
- 数字、浮点、`time.Time`、`[]byte` 收到空串时忽略该值（不修改字段、不报错）。
- 切片由全部输入值构建；空元素变成零值（`""`、`0`、`false`、零值时间）。
- `[]byte` 按单个字符串处理，不按数字列表处理。
- nil interface 取首值作为字符串；非 nil interface 通过创建动态类型的副本进行解码；嵌套 key 要求动态类型是 struct/slice/map/array。
- `required` 检查输入：key 缺失、value 列表为空或所有 value 都为空时返回 `EmptyFieldError`；显式 `0` 或 `false` 视为已提供。
- `WithZeroEmpty(true)` 会把空的数字、时间、`[]byte` 输入变成字段零值；指针字段会指向零值。
- `encoding.TextUnmarshaler` 接收首个输入 value（包括空串）；`Marshaler`/`Unmarshaler` 优先于 Text 接口。
- `time.Duration` 接受 `1s`、`1m30s`、`500ms` 等字符串，也接受数字纳秒（`1500000000`）。

编码：

- nil 指针、nil interface、nil/空切片、nil/空 map、长度为 0 的数组都不产生 key。
- `[]byte` 编码为一个字符串 value。
- `time.Time` 使用配置的 layout；没有 `omitempty` 时零值时间也会编码。
- `omitempty` 跳过 nil 和零值；空切片/map 本来就不会产生输出。
- 匿名内嵌结构体扁平化到父级 key。

## 支持的类型

| 类别 | 类型 |
|------|------|
| 基础类型 | `string`、`bool`、`int*`、`uint*`、`float32`、`float64` |
| 指针 | 任意支持类型的指针（含多级指针） |
| 容器 | 切片、数组、map、嵌套结构体 |
| 时间 | `time.Time`、`*time.Time`、`time.Duration`、`*time.Duration` |
| 字节 | `[]byte` 以及元素类型恰好为 `byte` 的命名类型 |
| Interface | `interface{}` 及其他 interface 字段 |
| 自定义 | 实现 `Marshaler` / `Unmarshaler` 或 `encoding.TextMarshaler` / `TextUnmarshaler` 的类型 |

interface 字段可以编码（使用动态值）和解码（nil interface 变成字符串；非 nil interface 通过动态类型解码）。函数、通道、复数、unsafe 指针不支持。

## 泛型

泛型类型透明支持；反射看到的是每个实例化类型（`Box[int]`、`Box[string]`），元数据缓存以 `reflect.Type` 为 key。

```go
type Box[T any] struct {
    Value T `form:"value"`
}

values, _ := form.Marshal(Box[int]{Value: 42}) // value=[42]

var box Box[string]
_ = form.Unmarshal(map[string][]string{"value": {"hello"}}, &box)
```

支持泛型别名（Go 1.24+）：

```go
type IntBox = Box[int]
```

## 限制

- 结构体 tag 名称不能包含当前 namespace 前缀、namespace 后缀、`[` 或 `]`，它们是路径分隔符；默认配置下即 `.`、`[`、`]`。
- map key 不能为空，也不能包含 `[` 或 `]`。
- 默认值不能包含 `|`（多值分隔符）或 `,`（tag 选项分隔符）。
- 默认值支持标量、指针、时间、`[]byte`、数组和标量切片字段；map 和结构体字段的默认值不会生效。
- namespace 后缀不能包含 namespace 前缀；这种配置无法无歧义解析，`Encode`/`Decode` 会直接返回错误。
- 同一个字段不能同时使用 `required` 和 `default`；解码时会返回配置错误。
- nil interface 只能接收字符串；要解码嵌套 key，interface 必须持有具体的 struct/slice/map/array 动态值。
- 未导出的内嵌值结构体和**非 nil** 的未导出内嵌指针结构体受支持；nil 未导出内嵌指针无法通过反射分配，如果有输入指向它会返回错误。
- 默认允许重复 tag 名（两个字段共享同一个 key）；启用 `WithRejectDuplicateTags(true)` 会返回 `ErrDuplicateTag`。
- 默认解码替换切片值；启用 `WithAppendSlices(true)` 会追加到已有切片。
- value receiver 的 `UnmarshalValues`/`UnmarshalText` 无法修改字段，默认静默无效果；启用 `WithRejectValueReceiverUnmarshalers(true)` 会返回错误。
- 顶层 struct 默认按字段编码，即使它实现了 `Marshaler`；启用 `WithTopLevelMarshaler(true)` 才会使用接口。
- 数组索引越界和非法切片索引会作为解码错误返回，而不是被忽略。
- 完全没有输入的指针结构体内部的默认值不会递归应用。
- 编码循环指针图时会返回最大深度错误，而不是栈溢出。

## 持续集成

仓库包含 `.github/workflows/ci.yml`，在 Go 1.24–1.27 上运行 `go vet`、`go test -race`、fuzz smoke test 和 golangci-lint。

- **GitHub**：将仓库推送/镜像到 GitHub 后，workflow 会自动运行。
- **Gitee Go**：Gitee 不会执行 GitHub Actions。仓库提供了 `.workflow/ci.yml` 模板；在仓库设置中启用 Gitee Go 即可（如果你的 Gitee Go 环境不支持 Go 1.24，请调整 `golangVersion`）。流水线执行同样的命令：
  ```bash
  go vet ./...
  go test -race -coverprofile=coverage.out ./...
  go test -run '^$' -fuzz=FuzzParseKeyPath -fuzztime=10s .
  go test -run '^$' -fuzz=FuzzDecode -fuzztime=10s .
  golangci-lint run ./...
  ```
- **本地**：`make ci` 会运行 fmt-check、vet、lint、race 测试和 fuzz smoke test。race detector 需要 cgo/gcc。

## 许可证

Apache-2.0
