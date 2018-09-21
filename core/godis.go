package core

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type dict map[string]*GodisObject
type GodisDB struct {
	Dict   dict //每一个DB都对应一个类型的Object
	Expire dict
	ID     int32
}
type GodisCommond struct {
	Name string
	Proc cmdFunc
}

type cmdFunc func(c *Client, s *Server)

type Server struct {
	Db               []*GodisDB               //DB 一整个指针数组  数组中每一个元素指向一个DB
	DbNum            int                      //DB num
	Start            int64                    //服务器创建的时间
	Aoffilename      string                   //Aof文件名
	RdbFileName      string                   //rdb文件名
	NextClientId     int32                    //为什么要有这个标记？？？
	SystemMemorySize int32                    //系统内存大小
	ClientNum        int32                    //链接的客户端计数
	Pid              int                      //PID进程ID·
	Commond          map[string]*GodisCommond //一个具体的命令对应着一个命令处理函数
	Dirty            bool                     //Dirty数据是否被污染
	AofBuf           []string                 //存储
}

//每一个client链接server之后都会创建一个Client结构
type Client struct {
	Cmd      *GodisCommond  //命令处理函数
	Argv     []*GodisObject //参数类型
	Argc     int            //参数数量
	Db       *GodisDB       //每一个client指向的DB
	QueryBuf string         //请求的buf
	Buf      string         //响应buf,返回给客户端的信息
}

func (s *Server) CreateClient(conn net.Conn) (c *Client) {
	c = new(Client)
	c.Argv = make([]*GodisObject, 5) //在这里创建5个是因为redis只有五种数据结构  我们这里只会使用一种
	c.Db = s.Db[0]                   //client.Db指向的是正在连接的db。如果有select切换操作，该指向也会随之变化。
	c.QueryBuf = ""
	return c
}

//客户端的请求都放在服务器创建的客户端中进行  服务端不需要感知具体是如何操作的  这些都有client来完成
func (c *Client) ReadQueryFromClient(conn net.Conn) (err error) {
	ibuf := make([]byte, 1024)
	_, err = conn.Read(ibuf)
	if err != nil {
		return err
	}
	//  log.Println("read读取客户端数据")
	// 拼接一个\n来组成一个完整的命令，在处理命令时根据\n即可判断

	str := string(ibuf)
	///log.Printf(str) //+++++++++++

	part := strings.Split(str, "\n") //将输出的内容根据\n切分成一个个片

	//在这里对协议进行实现
	c.QueryBuf = CmdProtocol(part[0]) //++++++++++++++++++++++++++++++++++
	//	c.QueryBuf = part[0]//获取最开始的以\n区分的一条命令-----------------
	return nil
}

func CmdProtocol(cmd string) (pro string) { //+++++++++++++++++++++++
	//set key value
	//rep := strings.Split(cmd, " ") //返回一个字符串数组
	rep := strings.FieldsFunc(cmd, unicode.IsSpace)
	//*3\r\n$3\r\nset\r\n$3\r\nkey\r\n$5\r\nvalue\r\n
	for k, v := range rep {
		//	v = strings.FieldsFunc(v, unicode.IsSpace)
		if k == 0 {
			//计算长度
			len := len(rep)
			pro = fmt.Sprintf("*%d\r\n", len)
		}
		pro += fmt.Sprintf("$%d\r\n%s\r\n", len(v), v)
	}
	return
}

func ProtocolToCmd(pro string) (argv []string, argc int) { //++++++++++++++++++++++
	parts := strings.Split(strings.Trim(pro, " "), "\r\n")
	//*3" "$3" "set" "$3" "key" "$5" "value" ""
	if len(parts) == 0 {
		return nil, 0
	}
	argc, err := strconv.Atoi(parts[0][1:])
	//	log.Print(argc)
	//	str := `abc def ghij    klmn
	//  123
	// 456`
	//	log.Printf("Fields are: %q", strings.FieldsFunc(str, unicode.IsSpace))
	if err != nil {
		return nil, 0
	}
	var vlen []int //定义一个记录参数长度的数组
	j := 0
	for _, value := range parts[1:] {
		if len(value) == 0 {
			continue
		}
		if value[0] == '$' {
			temp, err := strconv.Atoi(value[1:])
			if err == nil {
				vlen = append(vlen, temp)
			}
		} else {
			if j < len(vlen) && vlen[j] == len(value) {
				argv = append(argv, value)
				j++
			}
		}

	}

	return argv, argc

}

//ProcessInputBuffer 处理客户端请求信息
func (c *Client) ProcessInputBuffer() {
	//首先处理命令参数
	//MustCompile类似于编译，但如果无法解析表达式，就会引发恐慌。它简化了保存已编译正则表达式的全局变量的安全初始化。
	//r := regexp.MustCompile("[^\\s]+")-------------------------------
	//Trim返回字符串s的一部分，删除了cutset中包含的所有前导和后导Unicode代码点。
	//FindString返回一个字符串，其中包含正则表达式s中最左边匹配项的文本。如果没有匹配，则返回值为空字符串，但如果正则表达式成功匹配空字符串，则返回值为空。如果需要区分这些情况，可以使用FindStringIndex或FindStringSubmatch。
	//FindAllString是FindString的“All”版本;它返回表达式的所有连续匹配部分，如包注释中的“all”描述所定义的那样。返回值为nil表示不匹配。
	//parts := r.FindAllString(strings.Trim(c.QueryBuf, " "), -1) //在这里的限制是每一个命令及参数之间严格按照set key value 的格式进行查找，参数分别代表将每一个参数进行返回处理 -----------------------
	//得到argc以及argv
	//argc, argv := len(parts), parts-----------------------------------
	argv, argc := ProtocolToCmd(c.QueryBuf) //++++++++++++++++++
	c.Argc = argc
	//为每一个参数确定对应OBJECT类型的对象
	j := 0
	for _, val := range argv {
		c.Argv[j] = CreatObject(ObjectTypeString, val)
		j++
	}
}
func (s *Server) ProcessClientRequests(c *Client) {
	//在这里首先得到具体的命令
	v := c.Argv[0].Ptr
	name, ok := v.(string)
	if !ok {
		log.Println("error cmd")
		os.Exit(1)
	}
	//然后查询commond表是否有命令
	cmd := lookCommond(name, s)
	if cmd != nil {
		c.Cmd = cmd
		//如果有则执行对应操作
		call(c, s)
	} else {
		addReply(c, CreatObject(ObjectTypeString, fmt.Sprintf("(error) ERR unknown command '%s'", name)))
	}
}

func lookCommond(name string, s *Server) (cmd *GodisCommond) {
	if cmd, ok := s.Commond[name]; ok {
		return cmd
	}
	return nil

}
func call(c *Client, s *Server) {
	//	 if c.Cmd=="get"{
	//调用get处理命令
	//	 }else if c.Cmd=="set"{
	//调用set处理命令
	//	}

	//在这里体现出了接口的好处，只要实现借口就可以实现对应的函数
	c.Cmd.Proc(c, s)
}

func SetCommond(c *Client, s *Server) {
	//获取出参数
	objkey := c.Argv[1]
	objval := c.Argv[2]
	if c.Argc != 3 {
		addReply(c, CreatObject(ObjectTypeString, "(error) ERR wrong number of arguments for 'set' command"))
		return
	}
	if stringkey, ok1 := objkey.Ptr.(string); ok1 {
		if stringvalue, ok2 := objval.Ptr.(string); ok2 {
			c.Db.Dict[stringkey] = CreatObject(ObjectTypeString, stringvalue)
			//将key-value插入到对应的Db的对应的位置
		}
	}
	addReply(c, CreatObject(ObjectTypeString, "OK"))
}

func addReply(c *Client, object *GodisObject) {
	c.Buf = object.Ptr.(string)
}

func GetCommond(c *Client, s *Server) {
	godiscmd := lookUpKey(c.Db, c.Argv[1])
	if godiscmd != nil {
		addReply(c, godiscmd)
	} else {
		addReply(c, CreatObject(ObjectTypeString, "nil"))
	}
}

func lookUpKey(db *GodisDB, key *GodisObject) (ret *GodisObject) {
	if ret, ok := db.Dict[key.Ptr.(string)]; ok {
		return ret
	}
	return nil

}
