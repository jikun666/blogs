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

	// go func() {
	// 	time.Sleep(2 * time.Millisecond)
	// 	mutex.Lock()
	// 	println("goroutine b get lock")
	// 	time.Sleep(3 * time.Second)
	// 	println("goroutine b release lock")
	// 	mutex.Unlock()

	// 	wg.Done()
	// }()

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
