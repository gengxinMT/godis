package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	flag.Parse()
	if flag.NArg() != 2 {
		panic("flag error!")
	}
	fmt.Println("this is client")
	addrAndPort := fmt.Sprintf("%s:%s", flag.Arg(0), flag.Arg(1))
	//ipandport := "127.0.0.1:9736"

	reader := bufio.NewReader(os.Stdin)

	tcpAddr, _ := net.ResolveTCPAddr("tcp4", addrAndPort)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	checkError(err, "Dial error")
	defer conn.Close()
	go gethello(conn)
	time.Sleep(time.Millisecond * 100)
	for {
		fmt.Print(addrAndPort + ">")
		text, _ := reader.ReadString('\n') //读到/n结束
		//message := strings.Trim(text, "\n") //清除文本中的换行符
		//	text = strings.Replace(text, "\n", "", -1)

		sendtoServer(conn, text)

		readbuf := make([]byte, 1024)
		length, err := conn.Read(readbuf)
		checkError(err, "read error")

		if length == 0 {
			fmt.Println(addrAndPort+">", "nil")
		} else {
			fmt.Println(addrAndPort+">", string(readbuf))
		}

	}

}
func gethello(conn *net.TCPConn) {
	buf := make([]byte, 30)
	conn.Read(buf)
	fmt.Print(string(buf))
}
func checkError(err error, info string) {
	if err != nil {
		log.Println("Error:" + info + "" + err.Error())
		os.Exit(1)
	}
}

func sendtoServer(conn net.Conn, message string) (n int, err error) {
	buf := []byte(message)
	n, err = conn.Write(buf)
	return n, err
}
