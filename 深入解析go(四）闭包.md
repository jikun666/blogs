一种常见的闭包：

```go
func f(i int) func() int {
    return func() int {
        i++
        return i
    }
}
```

变量 i 是函数 f 中的局部变量，如果这个变量是在函数 f 的栈中分配就会出问题，因为函数 f 返回以后，对应的栈就失效了，则变量 i 就取不到了。

所以闭包环境中引用的变量不能够在栈上分配。


**逃逸分析**

直接来看一段代码：

```go
package main

type Cursor struct {
	X int
}

func f() *Cursor {
	var c Cursor
	c.X = 500
	return &c
}

func main() {
	f()
}
```

f 函数在 C 语言中是不允许的，但是 golang 是可以的，编译器会自动识别出这种情况并在堆上分配 C 的内存，而不是在函数 f 的栈上。

使用 go build 编译：

```
go build --gcflags=-m main.go
```

可以看到输出：

```
# command-line-arguments
./main.go:9:6: can inline f
./main.go:15:6: can inline main
./main.go:16:3: inlining call to f
./main.go:10:6: moved to heap: c
```

最后一行 `./main.go:10:6: moved to heap: c` 表示变量 c 逃逸了，逃到了堆上。

其实闭包是用**结构体**实现的，比如本文的第一段代码，就可以用如下结构体表示：

```go
type Closure struct {
    F func()() 
    i *int
}
```