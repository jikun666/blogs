

一个进程在内存里有三部分的数据，就是"代码段"、"堆栈段"和"数据段"。


# fork()

fork() 通过拷贝当前进程创建一个子进程。子进程和父进程使用相同的代码段；子进程复制父进程的堆栈段和数据段。


这样，父进程的所有数据都可以留给子进程，但是，子进程一旦开始运行，虽然它继承了父进程的一切数据，但实际上数据却已经分开，相互之间不再有影响了，也就是说，它们之间不再共享任何数据了。

它们再要交互信息时，只有通过进程间通信来实现。

# exec()

exec() 函数负责读取可执行文件并将其载入地址空间开始运行。

一个进程一旦调用exec类函数，它本身就"死亡"了，系统把代码段替换成新的程序的代码，废弃原有的数据段和堆栈段，并为新程序分配新的数据段与堆栈段，唯一留下的，就是进程号。


即，对系统而言，还是同一个进程，不过已经是另一个程序了。

# 平滑重启


如果程序想启动另一程序的运行，但自己仍想继续运行，就先 fork() 出一个子进程。然后在子进程中 exec() 另一程序。

这样便能使“父进程”继续执行，“子进程”执行另一程序了。

更重要的，通过 fork 创建子进程的方式可以实现父子进程监听相同的端口。



所谓的平滑重启即是如此原理。可参考 https://github.com/facebookarchive/grace.git。

fork 出的子进程监听继承过来的端口，然后继续通过 `os.Getppid()` 获取父进程的进程号，向父进程发送 kill 命令（`syscall.Kill(ppid, syscall.SIGTERM)`）。父进程收到信号后会执行优雅退出。


在上面的仓库中，并没有显式调用 fork+exec，上面通过环境变量传递监听描述字完成监听端口的继承，就不需要其他的 fork 过程了。

golang 如何判断是子进程还是父进程？

在fork之前，在环境变量中写入值，比如当前进程监听的文件描述符数量。

在进程起来时，通过上面那个环境变量的值判断是不是子进程。如果那个环境变量是空的，证明是父进程，不空则为子进程。