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

golang 中读锁的实现：

```golang
// RLock locks rw for reading.
//
// It should not be used for recursive read locking; a blocked Lock
// call excludes new readers from acquiring the lock. See the
// documentation on the RWMutex type.
func (rw *RWMutex) RLock() {
	if race.Enabled {
		_ = rw.w.state
		race.Disable()
	}
	if atomic.AddInt32(&rw.readerCount, 1) < 0 {
		// A writer is pending, wait for it.
		runtime_SemacquireMutex(&rw.readerSem, false, 0)
	}
	if race.Enabled {
		race.Enable()
		race.Acquire(unsafe.Pointer(&rw.readerSem))
	}
}
```

注释中有写到当读锁未释放时，将会阻塞其他读锁获取者。

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

golang 中写锁的实现：

```golang
// Lock locks rw for writing.
// If the lock is already locked for reading or writing,
// Lock blocks until the lock is available.
func (rw *RWMutex) Lock() {
	if race.Enabled {
		_ = rw.w.state
		race.Disable()
	}
	// First, resolve competition with other writers.
	rw.w.Lock()
	// Announce to readers there is a pending writer.
	r := atomic.AddInt32(&rw.readerCount, -rwmutexMaxReaders) + rwmutexMaxReaders
	// Wait for active readers.
	if r != 0 && atomic.AddInt32(&rw.readerWait, r) != 0 {
		runtime_SemacquireMutex(&rw.writerSem, false, 0)
	}
	if race.Enabled {
		race.Enable()
		race.Acquire(unsafe.Pointer(&rw.readerSem))
		race.Acquire(unsafe.Pointer(&rw.writerSem))
	}
}
```

注释中有写到如果有读锁和写锁的未释放时，将会阻塞等待。