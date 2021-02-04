哨兵（sentinel）是 redis 的一种故障转移手段。当 redis 处于复制模式下时，主节点故障，哨兵会挑选出某一个从节点成为主节点（从节点执行 slaveof no one 命令，即成为主节点），当原来的主节点重新上线时，哨兵会将其加入为从节点。

哨兵是由一个或多个哨兵实例组成的系统，也称哨兵系统。而每一个哨兵实例本质上是一个运行在特殊模式下的 redis 服务器。

```
redis-sentinel /path/to/your/sentinel.conf
```

或者


```
redis-server /path/to/your/sentinel.conf --sentinel
```

在配置文件中，可以指定需要监视的 redis 服务器（通常是主节点）。

```
sentinel monitor master 127.0.0.1 6379
```

初始化 Sentinel 的最后一步是创建连向被监视主服务器的网络连接，Sentinel 将成为主节点的客户端，它可以向主节点发送命令，并从命令回复中获取相关的信息。


对于被 Sentinel 监视的主节点来说，Sentinel 会创建两个连向主节点的异步网络连接：

1. 命令连接。这个连接专门用于向主节点发送命令，并接收命令回复；
2. 订阅连接，这个连接专门用于订阅主节点的 \_\_sentinel__:hello 频道。


## 获取主节点信息


Sentinel 默认会以每十秒一次的频率，通过**命令连接**向被监视的主节点发送 INFO 命令，并通过分析 INFO 命令的回复来获取主节点的当前信息比如 runID。（INFO 命令不仅会返回当前主节点的信息，也会返回从节点的地址等基本信息）


## 获取从节点信息


Sentinel 通过主节点获取到从节点的地址后，Sentinel 会创建连接到从节点的命令连接和订阅连接。

创建连接后，和主节点类似，sentinel 也会以默认每十秒一次的频率通过**命令连接**向从节点发送 INFO 命令，并获得从节点的信息比如 runID。


## 向主节点和从节点发送信息


默认情况下，sentinel 会以每两秒一次的频率，通过**命令连接**向所有被监视的主节点和从节点发送 PUBLISH 命令：

```
PUBLISH __sentinel__:hello "<s_ip>,<s_port>,<s_runid>,<s_epoch>,<m_name>,<m_ip>,<m_port>,<m_epoch>"
```

其中 s_ 开头的记录的是 sentinel 本身的信息。

s_runid 即哨兵的运行时id，用于在订阅消息时区分是否是自己发送的消息还是其他哨兵发送的消息。

s_epoch 即当前的配置纪元，用于 sentinel 选举 leader。

m_ 开头的记录的是主节点的信息（如果该节点是从节点，则记录的是从节点正在复制的主节点的信息）。


## 接收来自主节点和从节点的频道信息


Sentinel 会通过**订阅连接**向主节点和从节点发送 SUBSCRIBE 命令：


```
SUBSCRIBE __sentinel__:hello
```

即对于每个 Sentinel 连接的 redis 服务器，Sentinel 通过**命令连接**向 `__sentinel__:hello` 频道发送信息，通过**订阅链接**从 `__sentinel__:hello` 接收信息。



这里订阅收到的消息正是 Sentinel 通过 PUBLISH 发送的消息。对于监视同一个 redis 服务器的多个 Sentinel 来说，一个 sentinel 发送的信息会被其他 sentinel 接收到。这些信息会被用于更新其他 sentinel 对发送信息 sentinel 的认知，也会被用于更新其他 sentinel 对被监视服务器的认知。

当 sentinel 接收订阅到的消息，解析其中的哨兵信息和主节点信息，当解析得到的 s_runid 字段和自身的 runid 相同，则证明是自身发送的消息，则忽略本次消息，否则是其他哨兵发送的消息，需要对相应主节点对实例结构进行更新。

因为一个 sentinel 可以通过分析接收到的频道信息获知其他 sentinel 的存在，并通过发送频道信息来让其他 sentinel 知道自己的存在，所以用户在使用 sentinel 时不需要提供各个 sentinel 的信息，监视同一个主节点的多个 sentinel 可以自动发现对方。


## 创建 sentinel 之间的命令连接

当 sentinel 发现来彼此之后，sentinel 之间会创建**命令连接**，实现主观下线和客观下线检测。


## 检测主观下线状态

默认情况下，sentinel 会以每秒一次的频率向所有与它创建了命令连接的实例（主节点、从节点以及其他 sentinel）发送 PING 命令，并通过实例返回的 PING 命令回复来判断实例是否在线。

Sentinel 根据配置文件中的 `down-after-milliseconds` 选项指定 sentinel 判断实例进入主观下线所需的时间长度。

如果一个实例在 `down-after-milliseconds` 毫秒内，连续向 sentinel 返回无效回复，那么 sentinel 会标识该实例进入主观下线状态。



## 检测客观下线


当 sentinel 将一个**主节点**判断为主观下线之后，为了确认该主节点是否真的下线，它会向同样监视这一主节点的其他 sentinel 进行询问，看他们是否也认为该主节点已经进入了下线状态，当 sentinel 从其他 sentinel 那里接收到足够多的下线判断后， sentinel 就会将**从节点**（为什么是从节点？怀疑是笔误，应该是主节点）判定为客观下线状态，并对主节点执行故障转移操作。



“足够多的下线判断” 这个具体数字可有启动时配置文件载入，比如启动时载入配置为

```
sentinel monitor master 127.0.0.1 6379 3
```


即包括当前 sentinel 在内，总共至少要有 3 个 sentinel 都认为主节点已下线，当前 sentinel 才会将主节点判断为客观下线。


## 选举领头 Sentinel

领头 sentinel 负责对主节点进行故障转移操作。选举过程与 raft 协议中的 leader election 过程类似。



当某个 sentinel 判定某个主节点主观下线后，该 sentinel 会成为 candidate 并开始竞选，即向其他 sentinel 发送竞选消息，如果其他 sentinel 和源 sentinel 在同一个选举纪元并且也判定那个主节点下线，则会回复源 sentinel 一个选票，当源 sentinel 获得超过半数 sentinel 数量的选票后，该 sentinel 将会成为 leader，并更新选举纪元（上文提到的 s_epoch）。



## 故障转移


领头 sentinel 会对已下线的主节点进行故障转移操作：

1. 在所有的从节点中挑选出一个作为主节点；
    
    有一套完整的筛选规则，总的来说就是挑选与主节点数据最接近的那个从节点（可通过复制偏移量比较获得）

2. 让其他从节点改为复制新的主节点；
3. 将旧主节点设置为从节点，当其重新上线后，将作为从节点继续工作。


