事发于压测过程。


```
panic: runtime error: invalid memory address or nil pointer dereference [recovered]
    panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x52a12d]

goroutine 76354 [running]:
encoding/json.(*encodeState).marshal.func1(0xc001d51858)
    /opt/golang/go1.13/src/encoding/json/encode.go:305 +0x9a
panic(0xbb19e0, 0x1345d50)
    /opt/golang/go1.13/src/runtime/panic.go:679 +0x1b2
encoding/json.(*encodeState).string(0xc00246c3f0, 0x0, 0x1d4, 0x1)
    /opt/golang/go1.13/src/encoding/json/encode.go:895 +0x5d
encoding/json.stringEncoder(0xc00246c3f0, 0xb5f4a0, 0xc001bca1c0, 0x98, 0xc001bc0100)
    /opt/golang/go1.13/src/encoding/json/encode.go:610 +0xd9
encoding/json.(*encodeState).reflectValue(0xc00246c3f0, 0xb5f4a0, 0xc001bca1c0, 0x98, 0xc001bc0100)
    /opt/golang/go1.13/src/encoding/json/encode.go:337 +0x82
encoding/json.interfaceEncoder(0xc00246c3f0, 0xb90900, 0xc001bca2a0, 0x94, 0xc001bc0100)
    /opt/golang/go1.13/src/encoding/json/encode.go:619 +0xac
encoding/json.mapEncoder.encode(0xd03520, 0xc00246c3f0, 0xb96cc0, 0xc0006dce70, 0x15, 0xb90100)
    /opt/golang/go1.13/src/encoding/json/encode.go:706 +0x351
encoding/json.(*encodeState).reflectValue(0xc00246c3f0, 0xb96cc0, 0xc0006dce70, 0x15, 0x100)
    /opt/golang/go1.13/src/encoding/json/encode.go:337 +0x82
encoding/json.(*encodeState).marshal(0xc00246c3f0, 0xb96cc0, 0xc0006dce70, 0xc001d50100, 0x0, 0x0)
    /opt/golang/go1.13/src/encoding/json/encode.go:309 +0x10b
encoding/json.Marshal(0xb96cc0, 0xc0006dce70, 0xc001e96cf0, 0xc0014c8708, 0x1, 0x1, 0xc0006dce70)
    /opt/golang/go1.13/src/encoding/json/encode.go:161 +0x52
```

在测试单个用例时未发现问题，在高并发的场景下出现上面的 panic，首先定位上述 panic 所在的代码：

```golang
fakeRes := &fakeResSt{
    Data:           res.GetData(),
    Ext:            res.GetExt(),
    CandidateList:  candidates,
    WReqlog:        res.GetWReqlog(),
}
resBytes, _ := json.Marshal(fakeRes)
// ......
```

在 json.Marshal 那一行出现，经过反复查找，发现原来 fakeRes 的 WReqlog 存在并发读写的情况，即上面的字段是读 res 的 WReqlog，还有一个 goroutine 正在写 res 的 WReqlog，完整代码如下：

```golang
func handle() {
    defer func {
        go func() {
            res.WReqlog = "xxxxxxx"
        }()
    }()
    // ......
}

func getCands() {
    defer func {
        fakeRes := &fakeResSt{
            Data:           res.GetData(),
            Ext:            res.GetExt(),
            CandidateList:  candidates,
            WReqlog:        res.GetWReqlog(),
        }
        resBytes, _ := json.Marshal(fakeRes)
    }()

    handle()
    // ......
}
```

在执行 getCands() 时，可能 handle 函数中的 defer 有一个 goroutine 还未完成 res 的 WReqlog 写过程，getCands 中的 defer 就读取 res 的 WReqlog，随即进行 json.Marshal 过程，触发 panic。

参考：https://segmentfault.com/a/1190000023283854