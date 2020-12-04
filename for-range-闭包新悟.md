本项目一共踩了两个和 for-range 相关的坑。

### 第一个

```golang
for adID, dspid := range ruleDict.Pdb {
    dspidInt, _ := strconv.ParseInt(dspid, 10, 64)
    pdbInfo = append(pdbInfo, &waxdspproto.PdbInfo{
        DspId: &dspidInt,
        AdId:  &adID,
    })
}
```

在拼装发往 wax-dsp 请求体的过程中，由于使用 pb 格式进行传输。故上面的 pdbInfo 数组中的元素字段使用指针。

然而在 for-range 结构中，adid 和 dspid 变量地址是固定的，所以对 adid 执行取址一直是固定的。

需要使用新的局部变量 catch 住 adid 这个变量。如下：

```golang
for adID, dspid := range ruleDict.Pdb {
    dspidInt, _ := strconv.ParseInt(dspid, 10, 64)
    adidCatch := adid
    pdbInfo = append(pdbInfo, &waxdspproto.PdbInfo{
        DspId: &dspidInt,
        AdId:  &adidCatch,
    })
}
```


注意如果 for-range 的变量是结构体指针类型，则不存在上述问题，如下所示：


```golang
for i, impItem := range param.Imp {
    imp[i] = &waxdspproto.IdxImpInfo{
        Position: &impItem.Position,
        Impid:    &impItem.Impid,
    }
}
```


### 第二个

```golang
func newPools(config cfg.RedisCfg) (pools []*Pool) {
	serverDB := config.ServerDB
	for _, node := range config.Nodes {
		redPool := &redis.Pool{
			MaxIdle:   config.MaxIdle,
			MaxActive: config.MaxActive,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", node)
			},
		}
        // ......
	}
	return
}
```

如上根据不同的 node 建立 redPool，然后最终发现所有的连接都连接的是 config.Nodes 数组中最后一个 node，原因就在于这个 node 是传给 Dial 函数的，即这层闭包只有用的都是 for-range 中的一个 node 变量，需修改为：

```golang
func newPools(config cfg.RedisCfg) (pools []*Pool) {
	serverDB := config.ServerDB
	for _, v := range config.Nodes {
                node := v // 使用局部变量 catch
		redPool := &redis.Pool{
			MaxIdle:   config.MaxIdle,
			MaxActive: config.MaxActive,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", node)
			},
		}
        // ......
	}
	return
}
```