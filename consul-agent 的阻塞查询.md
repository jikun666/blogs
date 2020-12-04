在调用索引服务时，使用的是 [go-pool](http://git.intra.weibo.com/adx/go-pool/-/tree/v1.4.0) 通过 consul 发现索引服务节点并建立连接池。

在 go-pool 模块中有如下一段代码：

```golang
consulConf := api.DefaultConfig()
consulConf.Address = c.consulAddr
client, err := api.NewClient(consulConf)
if err != nil {
    c.log.Errorf("new consul client failed,err:%v", err)
    return nil, err
}

servicesEntries, meta, err := client.Health().Service(c.serviceName, c.tag, true, &api.QueryOptions{WaitIndex: c.index})
if err != nil {
    return nil, err
}
if len(servicesEntries) == 0 {
    err = errors.New("not find, please check your service name")
}

c.index = meta.LastIndex
```

上面调用的是 consul 提供的官方的 [api 库](https://github.com/hashicorp/consul/tree/v1.5.0/api)，其中的 `&api.QueryOptions{WaitIndex: c.index}` 是一个触发阻塞查询（blocking queries）的参数。见如下参数注释：

```golang
// WaitIndex is used to enable a blocking query. Waits
// until the timeout or the next index is reached
WaitIndex uint64

// WaitHash is used by some endpoints instead of WaitIndex to perform blocking
// on state based on a hash of the response rather than a monotonic index.
// This is required when the state being blocked on is not stored in Raft, for
// example agent-local proxy configuration.
WaitHash string

// WaitTime is used to bound the duration of a wait.
// Defaults to that of the Config, but can be overridden.
WaitTime time.Duration
```

WaitIndex - 类似于一个状态索引，只有当 WaitIndex > 0 时才触发阻塞查询。

WaitHash - 用于某些场景下 WaitIndex 的替代（不在此处讨论）。

WaitTime - 阻塞查询的超时时间。

阻塞调用是一个长轮询（long polling)，用于获取 consul-agent 中的状态变化。

一个阻塞查询的 HTTP 响应头会有一个 X-Consul-Index，代表了本次请求时 consul-agent 状态的唯一标识。

在 HTTP 请求时可以设置 X-Consul-Index 请求头，代表请求时当前客户端的状态。上面的 `&api.QueryOptions{WaitIndex: c.index}` 其实就是设置了该请求头。


consul-agent 收到 HTTP 查询请求时，如果发现 X-Consul-Index 请求头与自己的 index 一致，则会 hang 住直到内部状态改变或者 到达 WaitTime 时间；如果不一致，则会立即根据当前的状态信息返回查询结果。两种情况都会在响应头 X-Consul-Index 中设置代表本次状态的 index，即上面代码中的 `meta.LastIndex`。

注意 consul-agent 无法保证提前返回时状态一定是改变了，对 consul-agent 中的状态信息进行幂等的写操作，有可能会提前返回并返回一个不一样的 index。

注意 index 值一定是递增的改变的，即新状态的 index 一定是大于旧状态的 index。


官网地址：https://www.consul.io/api-docs/features/blocking