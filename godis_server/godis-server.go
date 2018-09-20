package main

import (
	"flag"
	"fmt"
	"godis/core"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var godis = new(core.Server)

const maxRead = 255

func main() {
	//	flag.Parse()
	argv := os.Args
	if argv[1] == "-v" || argv[1] == "-version" {
		version()
	}
	if argv[1] == "-h" || argv[1] == "-help" {
		usage()
	}
	flag.Parse()
	hostandPort := fmt.Sprintf("%s:%s", flag.Arg(0), flag.Arg(1))
	listener := initServer(hostandPort)

	//监听退出信号
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

	go sigHandler(c)

	defer listener.Close()
	for {
		conn, err := listener.Accept()
		//	log.Println("等待建连")
		///sayhello(conn)
		checkError(err, "Accept:")
		go connectionHandler(conn)

	}

}

func initServer(hostandPort string) *net.TCPListener {
	serveraddr, err := net.ResolveTCPAddr("tcp", hostandPort) //指定服务器地址和端口
	checkError(err, "Resolving address:port failed: '"+hostandPort+"'")
	listener, err := net.ListenTCP("tcp", serveraddr)
	checkError(err, "listen error:")
	println("Listening to :", listener.Addr().String())

	godis.Pid = os.Getpid()
	godis.DbNum = 16
	initDb()
	godis.Start = time.Now().UnixNano() / 1000000 //以毫秒为单位获取当前时间戳

	getCommond := &core.GodisCommond{Name: "get", Proc: core.GetCommond}
	setCommond := &core.GodisCommond{Name: "set", Proc: core.SetCommond}

	//godis.Commond指向yigemao中的命令处理结构体 所以在这每一个命令对应一个core中的命令处理函数
	godis.Commond = map[string]*core.GodisCommond{
		"get": getCommond,
		"set": setCommond, //在这里只需要小写就可以
	}

	return listener
}
func initDb() {
	//在这里DB是一个字符串数组，数组的每一个元素是指向一个具体DB的指针
	godis.Db = make([]*core.GodisDB, godis.DbNum)
	for i := 0; i < godis.DbNum; i++ { //为数组中的每一个元素进行初始化
		godis.Db[i] = new(core.GodisDB)
		godis.Db[i].Dict = make(map[string]*core.GodisObject, 100) //为每一个元素的类型创建一块空间
		//godis.Db[i].Dict = make(map[list]*core.GodisObject,100)
	}
}
func connectionHandler(conn net.Conn) {
	//在这里创建一个client结构  用来保存记录链接等信息
	c := godis.CreateClient(conn)
	connfrom := conn.RemoteAddr().String()
	println("Connection from:", connfrom)
	sayhello(conn)
	for {
		//	log.Println("for循环等待接收请求")
		//	接受请求
		err := c.ReadQueryFromClient(conn) //不能在参数中传递readbuf因为会很容易溢出
		switch err {
		case nil:

			//初始化客户端buf信息
			c.ProcessInputBuffer()
			//对命令进行处理
			godis.ProcessClientRequests(c)
			//将请求返回给客户端
			writeResponseToClient(conn, c)
		default:
			log.Println("Closed connection :", connfrom)
			return
		}
	}
	//time.Sleep(time.Second * 1)
}

//read读取客户端数据
//2018/09/14 10:39:16 心跳机制给客户端写数据
//2018/09/14 10:39:16 client diedRT
//2018/09/14 10:39:17 for循环等待接收请求
//读客户端请求内容

func sayhello(conn net.Conn) {
	obuf := []byte{'L', 'e', 't', '\'', 's', ' ', 'G', 'O', '!', '\n'}
	wrote, err := conn.Write(obuf)
	checkError(err, "Write: wrote "+string(wrote)+" bytes.")

}

//将客户端的响应返回给客户端是服务器的功能  所以这个函数要放在这里
func writeResponseToClient(conn net.Conn, c *core.Client) {
	conn.Write([]byte(c.Buf))
}
func checkError(error error, info string) {
	if error != nil {
		panic("ERROR:" + info + "" + error.Error())
	}

}
func sigHandler(c chan os.Signal) {
	for s := range c {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			exitHandler()
		default:
			log.Println("signal ", s)
		}
	}
}
func exitHandler() {
	fmt.Println("exiting smoothly ...")
	fmt.Println("bye ")
	os.Exit(0)
}
func version() {
	println("Godis server v=0.0.1 sha=xxxxxxx:001 malloc=libc-go bits=64 ")
	os.Exit(0)
}
func usage() {
	println("Usage: ./godis-server [/path/to/redis.conf] [options]")
	println("       ./godis-server - (read config from stdin)")
	println("       ./godis-server -v or --version")
	println("       ./godis-server -h or --help")
	println("Examples:")
	println("       ./godis-server (run the server with default conf)")
	println("       ./godis-server /etc/redis/6379.conf")
	println("       ./godis-server --port 7777")
	println("       ./godis-server --port 7777 --slaveof 127.0.0.1 8888")
	println("       ./godis-server /etc/myredis.conf --loglevel verbose")
	println("Sentinel mode:")
	println("       ./godis-server /etc/sentinel.conf --sentinel")
	os.Exit(0)
}
