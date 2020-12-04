项目使用 [redigo v2.0.0](https://github.com/gomodule/redigo/tree/v2.0.0) 作为连接 redis 的 client 实现。

在初始化的时候有 5 个需要注意的参数从源码中贴出如下：

```golang
// Maximum number of idle connections in the pool.
MaxIdle int

// Close connections after remaining idle for this duration. If the value
// is zero, then idle connections are not closed. Applications should set
// the timeout to a value less than the server's timeout.
IdleTimeout time.Duration

// Close connections older than this duration. If the value is zero, then
// the pool does not close connections based on age.
MaxConnLifetime time.Duration

// Maximum number of connections allocated by the pool at a given time.
// When zero, there is no limit on the number of connections in the pool.
MaxActive int

// If Wait is true and the pool is at the MaxActive limit, then Get() waits
// for a connection to be returned to the pool before returning.
Wait bool
```

MaxIdle - 最大闲置连接数。即在没有请求的情况下，设置的连接数。

IdleTimeout - 闲置连接的超时时间。即一个连接一直处于闲置状态的寿命。0 代表永远不关闭。这个值应该比 redis 服务器的超时时间设置得短，否则可能连接服务器都超时了，该闲置连接还没关闭。

MaxConnLifetime - 连接的寿命。0 代表永不关闭。

MaxActive - 能同时存在的最大连接数。0 代表无限制，即只要调用 pool.Get()，就一定能返回连接，不管是不是新创建的还是原来就已存在的闲置连接。

Wait - 只在设置了 MaxActive 值的时候，这个参数才有意义。

当调用 pool.Get() 时，如果 Wait 为 true ，并且连接数到达 MaxActive 时，将不能再创建新的连接，阻塞等待其他的连接释放才可返回，源码实现如下：


```golang
// Handle limit for p.Wait == true.
if p.Wait && p.MaxActive > 0 {
    p.lazyInit()
    if ctx == nil {
        <-p.ch
    } else {
        select {
        case <-p.ch:
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
}
```

如果 Wait 为 false ，并且连接数到达 MaxActive 时，将直接返回连接池已用完的错误，源码实现如下：

```golang
// Handle limit for p.Wait == false.
if !p.Wait && p.MaxActive > 0 && p.active >= p.MaxActive {
    p.mu.Unlock()
    return nil, ErrPoolExhausted
}
```