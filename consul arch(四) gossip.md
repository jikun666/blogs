Gossip 是一种去中心化、最终一致性的协议。


consul 使用 gossip 来管理集群节点。注意和 raft 的区别，consul 使用 raft 管理 consul servers 之间的状态同步问题，是个强一致性协议；而 gossip 则实现了 consul cluster 中所有节点发现、失效探测等状态同步问题。

注意，既然 gossip 是最终一致性，即在某个时间点，会出现节点数据不同步的情况。

consul 的 gossip 基于 [serf 库](https://github.com/hashicorp/serf) 和 [memberlist 库](https://github.com/hashicorp/memberlist) 实现 gossip。

注意，consul 中有两种 gossip pool: LAN gossip 和 WAN gossip。其中 LAN gossip 适用于同一个 Datacenter 中的节点，故 consul server 和 consul client 都会进行初始化并运行；而 WAN gossip 适用于不同 Datacenters 中的 consul servers 之间，即 consul server 相比于 consul client 会有额外的 WAN gossip 逻辑。 

---

### 基本过程

gossip 每个节点共有 3 种状态：alive、suspect、dead

alive - 节点的正常运行状态；

suspect - 若探测某节点失败，则该节点状态在本地置为“可疑”；

dead - 当“可疑”超时后，本地会将该可疑节点转为 dead，并广播该信息；

注意，当“可疑节点”自身收到自己可疑的广播消息时，节点可以作出“反驳（refute）”，并且广播自己为 alive 的信息。

更具体的说：

1. 如果节点B无法被对节点 A 出的探测消息进行响应，或者响应超时，它会被节点 A 标为 suspect, 如果 suspect 持续一段时间（或它收到足够多的其它节点关于B的SuspectMsg），节点A会在集群中广播SuspectMsg，告知集群中的其它节点，节点B很可疑；
2. 如果B收到了针对它的 SuspectMsg，这显然是对它的不利言论，B可以通过发送 AliveMsg 告知对方, “I’m alive”。那么在对方节点看来 B 的 state 从suspect 变为 alive
3. 如果一段时间内，B 的状态仍然是 suspect, 那么对节点 A 而言，B 的状态会被置为 dead

![](images/consul_arch/gossip_node.drawio.png)

---


这里重点关注 memberlist 库的实现。


`github.com/hashicorp/memberlist/memberlist.go`

```golang
func Create(conf *Config) (*Memberlist, error) {
    m, err := newMemberlist(conf)
    if err != nil {
        return nil, err
    }
    if err := m.setAlive(); err != nil {
        m.Shutdown()
        return nil, err
    }
    m.schedule()
    return m, nil
}
```

在 `newMemberlist` 中开始监听 tcp / udp 端口：

```golang
func newMemberlist(conf *Config) (*Memberlist, error) {
    // ...
    go m.streamListen()
    go m.packetListen()
    go m.packetHandler()
    return m, nil
}


// tcp 监听
func (m *Memberlist) streamListen() {
    for {
        select {
        case conn := <-m.transport.StreamCh():
            go m.handleConn(conn)

        case <-m.shutdownCh:
            return
        }
    }
}

// udp监听
func (m *Memberlist) packetListen() {
    for {
        select {
        case packet := <-m.transport.PacketCh():
            m.ingestPacket(packet.Buf, packet.From, packet.Timestamp)

        case <-m.shutdownCh:
            return
        }
    }
}

// 事件监听，处理其他节点传递过来的数据包信息（udp)
func (m *Memberlist) packetHandler() {
    for {
        select {
        case <-m.handoffCh:
            for {
                // ...
                switch msgType {
                case suspectMsg:
                    m.handleSuspect(buf, from)
                case aliveMsg:
                    m.handleAlive(buf, from)
                case deadMsg:
                    m.handleDead(buf, from)
                case userMsg:
                    m.handleUser(buf, from)
                default:
                    m.logger.Printf("[ERR] memberlist: Message type (%d) not supported %s (packet handler)", msgType, LogAddress(from))
                }
            }
        // ...
        }
    }
}
```

在 `schedule` 使用了 3 个 goroutinue 用来分别执行 long running task:

```golang
func (m *Memberlist) schedule() {
    // ...

    if m.config.ProbeInterval > 0 {
        t := time.NewTicker(m.config.ProbeInterval)
        go m.triggerFunc(m.config.ProbeInterval, t.C, stopCh, m.probe)
        m.tickers = append(m.tickers, t)
    }

    if m.config.PushPullInterval > 0 {
        go m.pushPullTrigger(stopCh)
    }
    
    if m.config.GossipInterval > 0 && m.config.GossipNodes > 0 {
        t := time.NewTicker(m.config.GossipInterval)
        go m.triggerFunc(m.config.GossipInterval, t.C, stopCh, m.gossip)
        m.tickers = append(m.tickers, t)
    }

    // ...
}
```

(1) `go m.triggerFunc(m.config.ProbeInterval, t.C, stopCh, m.probe)` 使用 `probeIndex` 变量轮询集群中的节点，通过 udp 方式发送心跳探测包，以此对节点进行失败探测。

```golang
func (m *Memberlist) probe() {
    // ...

    // 轮询集群中的一个节点
    node = *m.nodes[m.probeIndex]
    m.probeIndex++

    // 探测节点
    m.probeNode(&node)
}
```

在执行 `m.probeNode(&node)` 时，若探测失败，则会针对探测的节点调用 `suspectNode` 方法，该方法根据收到的 suspect 信息进行相关处理:

<!-- 该方法会增加节点的“怀疑数”，当“怀疑数”达到阈值时可能会转化为失败节点（dead)： -->

```golang
// suspectNode is invoked by the network layer when we get a message
// about a suspect node
func (m *Memberlist) suspectNode(s *suspect) {
    // ...

    // 若 s.Node 已存在失败定时器
    if timer, ok := m.nodeTimers[s.Node]; ok {
        // 增加 s.Node 的 suspect 计数
        // 返回是否需要继续向其他节点传递该 suspect 信息
        if timer.Confirm(s.From) {
            m.encodeAndBroadcast(s.Node, suspectMsg, s)
        }
        return
    }
    

    if state.Name == m.config.Name {
        // 如果该节点是自身，则需要反驳该 suspect 信息，即广播 alive 数据包
        m.refute(state, s.Incarnation)
        m.logger.Printf("[WARN] memberlist: Refuting a suspect message (from: %s)", s.From)
        return // Do not mark ourself suspect
    } else {
        // 否则，继续广播该 suspect 信息
        m.encodeAndBroadcast(s.Node, suspectMsg, s)
    }

    // 否则，插入失败定时器
    min := suspicionTimeout(m.config.SuspicionMult, n, m.config.ProbeInterval)
    max := time.Duration(m.config.SuspicionMaxTimeoutMult) * min
    fn := func(numConfirmations int) { // 对 s.Node 确认为 suspect 的数量
            // ...
            d := dead{Incarnation: state.Incarnation, Node: state.Name, From: m.config.Name}
            m.deadNode(&d) // 转化为失败节点
        }
    }
    m.nodeTimers[s.Node] = newSuspicion(s.From, k, min, max, fn)
}
```


(2) `go m.pushPullTrigger(stopCh)`  定期随机选择 1 个节点，通过 tcp 传输方式与其做全量数据交换，加速集群内数据一致性收敛。

```golang
func (m *Memberlist) pushPullTrigger(stop <-chan struct{}) {
    interval := m.config.PushPullInterval

    // 随机错开一段时间
    randStagger := time.Duration(uint64(rand.Int63()) % uint64(interval))
    select {
    case <-time.After(randStagger):
    case <-stop:
        return
    }

    // 根据节点数动态间隔
    for {
        tickTime := pushPullScale(interval, m.estNumNodes())
        select {
        case <-time.After(tickTime):
            m.pushPull()
        case <-stop:
            return
        }
    }
}

func (m *Memberlist) pushPull() {
    // 随机选取 1 个节点
    m.nodeLock.RLock()
    nodes := kRandomNodes(1, m.nodes, func(n *nodeState) bool { 
        return n.Name == m.config.Name ||
            n.State != StateAlive
    })
    m.nodeLock.RUnlock()

    // If no nodes, bail
    if len(nodes) == 0 {
        return
    }
    node := nodes[0]

    // 全量数据交换
    if err := m.pushPullNode(node.FullAddress(), false); err != nil {
        m.logger.Printf("[ERR] memberlist: Push/Pull with %s failed: %s", node.Name, err)
    }
}
```

(3) `go m.triggerFunc(m.config.GossipInterval, t.C, stopCh, m.gossip)` 随机选取集群内 k 个节点，采用 udp 传输方式发送当前节点状态以及用户自定义数据。

```golang
func (m *Memberlist) gossip() {
    // 选取 alive、suspect 以及最近变成 dead 的 k 个节点
    kNodes := kRandomNodes(m.config.GossipNodes, m.nodes, func(n *nodeState) bool {
        if n.Name == m.config.Name {
            return true
        }

        switch n.State {
        case StateAlive, StateSuspect:
            return false

        case StateDead:
            return time.Since(n.StateChange) > m.config.GossipToTheDeadTime

        default:
            return true
        }
    })

    for _, node := range kNodes {
        addr := node.Address()
        // ... send message ...
    }
}
```

---

参考：
- https://www.consul.io/docs/architecture/gossip
