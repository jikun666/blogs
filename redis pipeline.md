可以一次性发送多条命令并在执行完后一次性将结果返回，pipeline 通过减少客户端与 redis 的通信次数来实现降低RTT(Round Trip Time 往返时间)。

Pipeline 实现的原理是队列，而队列是先进先出的，这样就保证数据的顺序性。

注意由于顺序执行命令并且整个 pipeline 的执行不是原子性的，所以在一个 pipeline 中先执行的命令造成的结果会影响后执行的语句，如下：

```go
package main

import (
	"fmt"
	"log"

	"github.com/gomodule/redigo/redis"
)

func echoReceive(res interface{}, err error) {
	if err != nil {
		fmt.Println(err)
	} else {
		if res != nil {
			fmt.Printf("---------  ")
			switch v := res.(type) {
			case []byte:
				fmt.Println(string(v))
			default:
				fmt.Println(v)
			}
		} else {
			fmt.Println(res)
		}

	}
}

func main() {
	c1, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		log.Fatalln(err)
	}
	defer c1.Close()

	c1.Send("Get", "my_test")
	c1.Flush()
	echoReceive(c1.Receive())

	c1.Send("Get", "my_test2")
	c1.Flush()
	echoReceive(c1.Receive())

	c1.Send("set", "bar", "foo")
	c1.Send("get", "bar") // 上条命令set 进去的也能读出来，前提是上条命令执行成功了
	c1.Send("Get", "my_test")
	c1.Send("Get", "my_test2")
	c1.Send("Get", "my_test3")
	c1.Flush()
	echoReceive(c1.Receive())
	echoReceive(c1.Receive())
	echoReceive(c1.Receive())
	echoReceive(c1.Receive())
	echoReceive(c1.Receive())
}
```

Pipeline每次只能作用在一个Redis节点。


我理解 pipeline 是半双工的，发送和接收不是同时的。而是 server 全部执行完后一次性返回。