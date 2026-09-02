V2Ray Transport 是 v2ray 发明的一组私有协议，并污染了其他协议的名称，如 clash 中的 `trojan-grpc`。

### 结构

```json
{
  "type": ""
}
```

可用的传输协议：

* HTTP
* WebSocket
* QUIC
* gRPC
* HTTPUpgrade
* XHTTP

!!! warning "与 v2ray-core 的区别"

    * 没有 TCP 传输层, 纯 HTTP 已合并到 HTTP 传输层。
    * 没有 mKCP 传输层。
    * 没有 DomainSocket 传输层。

!!! note ""

    当内容只有一项时，可以忽略 JSON 数组 [] 标签。

### HTTP

```json
{
  "type": "http",
  "host": [],
  "path": "",
  "method": "",
  "headers": {},
  "idle_timeout": "15s",
  "ping_timeout": "15s"
}
```

!!! warning "与 v2ray-core 的区别"

    不强制执行 TLS。如果未配置 TLS，将使用纯 HTTP 1.1。

#### host

主机域名列表。

如果设置，客户端将随机选择，服务器将验证。

#### path

!!! warning

    V2Ray 文档称服务端和客户端的路径必须一致，但实际代码允许客户端向路径添加任何后缀。
    sing-box 使用与 V2Ray 相同的行为，但请注意，该行为在 `WebSocket` 和 `HTTPUpgrade` 传输层中不存在。

HTTP 请求路径

服务器将验证。

#### method

HTTP 请求方法

如果设置，服务器将验证。

#### headers

HTTP 请求的额外标头

如果设置，服务器将写入响应。

#### idle_timeout

在 HTTP2 服务器中：

指定闲置客户端应在多长时间内使用 GOAWAY 帧关闭。PING 帧不被视为活动。

在 HTTP2 客户端中：

如果连接上没有收到任何帧，指定一段时间后将使用 PING 帧执行健康检查。需要注意的是，PING 响应被视为已接收的帧，因此如果连接上没有其他流量，则健康检查将在每个间隔执行一次。如果值为零，则不会执行健康检查。

默认使用零。

#### ping_timeout

在 HTTP2 客户端中：

指定发送 PING 帧后，在指定的超时时间内必须接收到响应。如果在指定的超时时间内没有收到 PING 帧的响应，则连接将关闭。默认超时持续时间为 15 秒。

### WebSocket

```json
{
  "type": "ws",
  "path": "",
  "headers": {},
  "max_early_data": 0,
  "early_data_header_name": ""
}
```

#### path

HTTP 请求路径

服务器将验证。

#### headers

HTTP 请求的额外标头

如果设置，服务器将写入响应。

#### max_early_data

请求中允许的最大有效负载大小。默认启用。

#### early_data_header_name

默认情况下，早期数据在路径而不是标头中发送。

要与 Xray-core 兼容，请将其设置为 `Sec-WebSocket-Protocol`。

它需要与服务器保持一致。

### QUIC

```json
{
  "type": "quic"
}
```

!!! warning "与 v2ray-core 的区别"

    没有额外的加密支持：
    它基本上是重复加密。 并且 Xray-core 在这里与 v2ray-core 不兼容。

### gRPC

!!! note ""

    默认安装不包含标准 gRPC (兼容性好，但性能较差), 参阅 [安装](/zh/installation/build-from-source/#构建标记)。

```json
{
  "type": "grpc",
  "service_name": "TunService",
  "idle_timeout": "15s",
  "ping_timeout": "15s",
  "permit_without_stream": false
}
```

#### service_name

gRPC 服务名称。

#### idle_timeout

在标准 gRPC 服务器/客户端：

如果传输在此时间段后没有看到任何活动，它会向客户端发送 ping 请求以检查连接是否仍然活动。

在默认 gRPC 服务器/客户端：

它的行为与 HTTP 传输层中的相应设置相同。

#### ping_timeout

在标准 gRPC 服务器/客户端：

经过一段时间之后，客户端将执行 keepalive 检查并等待活动。如果没有检测到任何活动，则会关闭连接。

在默认 gRPC 服务器/客户端：

它的行为与 HTTP 传输层中的相应设置相同。

#### permit_without_stream

在标准 gRPC 客户端：

如果启用，客户端传输即使没有活动连接也会发送 keepalive ping。如果禁用，则在没有活动连接时，将忽略 `idle_timeout` 和 `ping_timeout`，并且不会发送 keepalive ping。

默认禁用。

### HTTPUpgrade

```json
{
  "type": "httpupgrade",
  "host": "",
  "path": "",
  "headers": {}
}
```

#### host

主机域名。

服务器将验证。

#### path

HTTP 请求路径

服务器将验证。

#### headers

HTTP 请求的额外标头。

如果设置，服务器将写入响应。

### XHTTP

XHTTP (SplitHTTP) 是一种将双向全双工流拆分为独立 HTTP 请求与长连接下载流的新型传输协议，支持多路复用、CDN 前置代理与流量伪装。

```json
{
  "type": "xhttp",
  "mode": "auto",
  "host": "example.com",
  "path": "/xhttp-path",
  "headers": {},
  "x_padding_bytes": "100-1000",
  "sc_max_each_post_bytes": 1000000,
  "sc_min_posts_interval_ms": 30,
  "xmux": {
    "max_connections": 3,
    "c_max_reuse_times": "0-0",
    "h_max_request_times": "600-900",
    "h_max_reusable_secs": "1800-3000"
  }
}
```

#### mode

工作模式。

可选值：

* `auto`：自动协商或流式/分片自适应（默认）。
* `packet-up`：数据包上行模式（分片上传，适用于无流式上传支持的 CDN）。
* `stream-up`：流式上行模式（持续长连接分块上传）。
* `stream-one`：单向全双工长流模式（类似 HTTP/2 双向流）。

#### host

HTTP 请求的主机域名。

#### path

HTTP 请求的基础路径。

#### headers

HTTP 请求的额外键值对标头。

#### x_padding_bytes

XHTTP 填充字节数范围，用于混淆特征。例如 `"100-1000"` 或单个数字。

#### sc_max_each_post_bytes

在分片上传（`packet-up` / `auto`）时每个 POST 请求的最大字节数。默认 `1000000`（约 1MB）。

#### sc_min_posts_interval_ms

分片上传请求之间的最小间隔（毫秒）。默认 `30`。

#### xmux

XHTTP 连接复用与生命周期管理配置：

* `max_connections`：最大连接并发数范围（例如 `3` 或 `"1-4"`）。
* `max_concurrency`：最大流并发数范围（与 `max_connections` 互斥）。
* `c_max_reuse_times`：底层 TCP 连接最大复用次数范围。
* `h_max_request_times`：HTTP 请求最大复用次数范围（默认 `"600-900"`）。
* `h_max_reusable_secs`：连接最大可复用存活时间（秒，默认 `"1800-3000"`）。

#### download

进阶独立下载通道配置（用于上下行分离或不同 CDN/节点加速配置）：

```json
{
  "download": {
    "server": "download.example.com",
    "server_port": 443,
    "tls": {
      "enabled": true,
      "server_name": "download.example.com"
    }
  }
}
```
