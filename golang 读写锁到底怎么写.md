对于 golang 中的 map 结构，并发读写的时候需要使用读写锁防止 panic，读map的时候使用读锁：

```golang
func (aec *avgEcpmClient) get(ctx context.Context, tagid string) (avgEcpm int64, err error) {
	aec.mutex.RLock()
	v, ok := aec.data[tagid]
	aec.mutex.RUnlock()

	if ok {
		avgEcpm = v
		return
	}
	err = errors.New("redis not found avgEcpm")
	return
}
```

写map的时候使用写锁：

```golang
func (aec *avgEcpmClient) loop(t time.Time) (err error) {
    // ......
    aec.mutex.Lock()
    aec.data = data
    aec.mutex.Unlock()
    // ......
}
```



实战:

```go
package main

import (
	"sync"
	"time"
)

func main() {
	mutex := sync.RWMutex{}
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		mutex.RLock()
		println("goroutine a get lock")
		time.Sleep(2 * time.Second)
		println("goroutine a release lock")
		mutex.RUnlock()

		wg.Done()
	}()

	go func() {
		time.Sleep(2 * time.Millisecond)
		mutex.RLock()
		println("goroutine c get lock")
		time.Sleep(5 * time.Millisecond)
		println("goroutine c release lock")
		mutex.RUnlock()

		wg.Done()
	}()

	wg.Wait()
}
```

输出
```
goroutine a get lock
goroutine c get lock // 和上面一行几乎同时输出
goroutine c release lock
goroutine a release lock
```

如上代码所示，goroutine a 加读锁之后，goroutine c 也能直接获取读锁。

再通过其他的实验总结：

加读锁之后，可以再获取读锁，但无法获取写锁；

加写锁之后，无法获得读锁和写锁。

注意，获取锁的goroutine会排队，比如 a 加了读锁，b 来获取写锁失败，然后 c 想再获取读锁也会失败，c 必须等写锁 b 获取之后并释放才能获得锁，如下：

```go
package main

import (
	"sync"
	"time"
)

func main() {
	mutex := sync.RWMutex{}
	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		mutex.RLock()
		println("goroutine a get lock")
		time.Sleep(2 * time.Second)
		println("goroutine a release lock")
		mutex.RUnlock()

		wg.Done()
	}()

	go func() {
		time.Sleep(2 * time.Millisecond)
		mutex.Lock()
		println("goroutine b get lock")
		time.Sleep(3 * time.Second)
		println("goroutine b release lock")
		mutex.Unlock()

		wg.Done()
	}()

	go func() {
		time.Sleep(5 * time.Millisecond)
		mutex.RLock()
		println("goroutine c get lock")
		time.Sleep(5 * time.Millisecond)
		println("goroutine c release lock")
		mutex.RUnlock()

		wg.Done()
	}()

	wg.Wait()
}
```

输出

```
goroutine a get lock
goroutine a release lock
goroutine b get lock // a获得锁2秒后才输出
goroutine b release lock
goroutine c get lock // b获得锁3秒后才输出
goroutine c release lock
```

