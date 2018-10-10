package proto

import (
	"bytes"
	"errors"
	"godis/util/bufio2"
	"io"
	"log"
	"strconv"
)

//编码
type EnCoder struct {
	bw  *bufio2.Writer
	Err error
}

const (
	// MaxBulkBytesLen 最大长度
	MaxBulkBytesLen = 1024 * 1024 * 512
	// MaxArrayLen 最大长度
	MaxArrayLen = 1024 * 1024
)

var (
	ErrBadArrayLen        = errors.New("bad array len")
	ErrBadArrayLenTooLong = errors.New("bad array len, too long")

	ErrBadBulkBytesLen        = errors.New("bad bulk bytes len")
	ErrBadBulkBytesLenTooLong = errors.New("bad bulk bytes len, too long")

	ErrBadMultiBulkLen     = errors.New("bad multi-bulk len")
	ErrBadMultiBulkContent = errors.New("bad multi-bulk content, should be bulkbytes")
)

func errorNew(msg string) error {
	return errors.New("error occur, msg ")
}

//在这里进行多层初始化是为了提高代码复用性
//++++++++++++++++++55555555555555555555
func NewEncoder(w io.Writer) *EnCoder {
	return NewEncoderBuffer(bufio2.NewWriterSize(w, 8192))
}
func NewEncoderSize(w io.Writer, size int) *EnCoder {
	return NewEncoderBuffer(bufio2.NewWriterSize(w, size))
}
func NewEncoderBuffer(bw *bufio2.Writer) *EnCoder {
	return &EnCoder{bw: bw}
}

//+++++++++++++++++++++++++1111111111111111111
func EncodeCmd(cmd string) ([]byte, error) {
	return EncodeBytes([]byte(cmd))
}

//++++++++++++++++++=222222222222222222
func EncodeBytes(cmd []byte) (buf []byte, err error) {
	//在这里将字符串解析成一个个字节 然后后续交给对应的函数进行处理
	//rep := strings.FieldsFunc(cmd, unicode.IsSpace)
	rep := bytes.Split(cmd, []byte(" "))
	if rep == nil {
		return nil, ErrorTrace(errorNew("encodebyte err"))
	}
	resp := NewMultReply(nil) //resp的类型是多条批量回复类型
	//遍rep得到每一个部分的内容
	for _, val := range rep {
		if len(val) > 0 {
			//在这里为每一个解析出来的部分创建一个resp对象
			resp.Array = append(resp.Array, NewBulkReply(val)) //这里是将每一条记录处理为批量回复类型
		}
	}
	return EncodeToBytes(resp)

}

//++++++++++++++++++++333333333333333333
func EncodeToBytes(resp *Resp) (buf []byte, err error) {
	var b = &bytes.Buffer{}
	//以上只是对客户端传进来的命令拆解为一个个的字符串 接下来进行正式的编码
	if err := EnCode(b, resp); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

//+++++++++++++++++++44444444444444444
func EnCode(w io.Writer, r *Resp) error {
	return NewEncoder(w).EnCode(r, true)
}

//+++++++++++++++++66666666666666666
func (e *EnCoder) EnCode(r *Resp, flush bool) error {
	if e.Err != nil {
		return ErrorTrace(e.Err)
	}
	//编码
	if err := e.encodeResp(r); err != nil {
		e.Err = err
	} else if flush {
		e.Err = ErrorTrace(e.bw.Flush())
	}
	return e.Err

}

//encodeResp 编码
//+++++++++++++++++++7777777777777777
func (e *EnCoder) encodeResp(r *Resp) error {
	//首先解析协议类型  奖类型先写入缓存区中

	if err := e.bw.WriteByte(byte(r.Type)); err != nil {
		return ErrorTrace(err)
	}
	//然后根据编码的类型确定处理的方法
	switch r.Type {
	case TypeString, TypeError, TypeInt:
		//在这里进行文本类型的编码
		return e.encodeTextBytes(r.Value)
	case TypeBulkReply:
		//在这里进行批量回复
		return e.encodeBulkBytes(r.Value)

	case TypeMultiReply:
		//在这里进行多条批量回复
		return e.encodeMulti(r.Array)
	default:
		return ErrorTrace(e.Err)
	}
}

//8888888888888888888888888
// encodeTextBytes encode text type
func (e *EnCoder) encodeTextBytes(b []byte) error {
	if _, err := e.bw.Write(b); err != nil {
		return ErrorTrace(err)
	}
	if _, err := e.bw.WriteString("\r\n"); err != nil {
		return ErrorTrace(err)
	}
	return nil
}

//88888888888888888888888888888888888
//$3\r\nset\r\n
func (e *EnCoder) encodeBulkBytes(b []byte) error {
	if b == nil {
		return e.encodeInt(-1)
	} else {
		if err := e.encodeInt(int64(len(b))); err != nil {
			return err
		}
		return e.encodeTextBytes(b)
	}
}

//888888888888888888888888888
func (e *EnCoder) encodeMulti(mulit []*Resp) error {
	if mulit == nil {
		return e.encodeInt(-1)
	} else {
		//得到长度
		if err := e.encodeInt(int64(len(mulit))); err != nil {
			return err
		}
		for _, v := range mulit {
			if err := e.encodeResp(v); err != nil {
				return err
			}
		}

	}
	return nil
}

//999999999999999999999
func (e *EnCoder) encodeInt(i int64) error {
	return e.encodeTextString(strconv.FormatInt(i, 10))
}

//1000000000000000
func (e *EnCoder) encodeTextString(s string) error {
	if _, err := e.bw.WriteString(s); err != nil {
		return ErrorTrace(err)
	}
	if _, err := e.bw.WriteString("\r\n"); err != nil {
		return ErrorTrace(err)
	}
	return nil
}

//encodeMultiBulk encode 多条批量回复
func (e *EnCoder) encodeMultiBulk(mulit []*Resp) error {
	if err := e.bw.WriteByte(byte(TypeMultiReply)); err != nil {
		return ErrorTrace(err)
	}
	return e.encodeMulti(mulit)

}

//解码+++++++++++++++++++++
type Decoder struct {
	br *bufio2.Reader

	Err error
}

//11111111111111111111111111
func DecodeFromBytes(cmd []byte) (*Resp, error) {
	return NewDecode(bytes.NewReader(cmd)).Decode()
}

//2222222222222222222222222
func NewDecode(r io.Reader) *Decoder {
	return NewDecodeBuffer(bufio2.NewReaderSize(r, 8192))
}

//33333333333333333333333333
func NewDecodeSize(br io.Reader, size int) *Decoder {
	return NewDecodeBuffer(bufio2.NewReaderSize(br, size))
}

//4444444444444444444444444
func NewDecodeBuffer(r *bufio2.Reader) *Decoder {
	return &Decoder{br: r}
}

//5555555555555555555555555
func Decode(br io.Reader) (*Resp, error) {
	return NewDecode(br).Decode()
}

//666666666666666666666666
func (dec *Decoder) Decode() (*Resp, error) {
	if dec.Err != nil {
		return nil, ErrorTrace(errorNew("Decode error"))
	}
	r, err := dec.decodeResp()
	if err != nil {
		//	return nil ,ErrorTrace(err)//--------------------这里注意错误的处理方式
		dec.Err = err

	}
	return r, dec.Err

}

//7777777777777777777777777777777777
//根据返回类型调用不同的解析实现
func (dec *Decoder) decodeResp() (*Resp, error) {
	b, err := dec.br.ReadByte()
	if err != nil {
		return nil, ErrorTrace(err)
	}
	r := &Resp{}
	r.Type = byte(b) //判断解析出来的协议类型
	switch r.Type {
	case TypeError, TypeInt, TypeString:
		r.Value, err = dec.decodeTextBytes()

	case TypeBulkReply:
		r.Value, err = dec.decodeBulkBytes()
	case TypeMultiReply:
		r.Array, err = dec.decodeArray()

	default:
		return nil, ErrorTrace(err)
	}
	return r, err
}

func (dec *Decoder) decodeTextBytes() ([]byte, error) {

	val, err := dec.br.ReadBytes('\n')
	//	log.Println(string(val))
	if err != nil {
		return nil, ErrorTrace(err)
	}
	//读取之后将内容提取出来
	if len := len(val) - 2; len < 0 || val[len] != '\r' {
		//	log.Println("decode failed")
		return nil, ErrorTrace(err)
	} else {
		//	log.Println(val[:len])
		return val[:len], nil
	}

}

func (dec *Decoder) decodeInt() (int64, error) {
	val, err := dec.br.ReadSlice('\n')
	if err != nil {
		return 0, ErrorTrace(err)
	}
	//读取之后将内容提取出来
	if len := len(val) - 2; len < 0 || val[len] != '\r' {
		return 0, ErrorTrace(err)
	} else {
		return Btoi64(val[:len])
	}
}
func Btoi64(b []byte) (int64, error) {
	if len(b) != 0 && len(b) < 10 {
		var neg, i = false, 0
		switch b[0] {
		case '-':
			neg = true
			fallthrough
		case '+':
			i++
		}
		if len(b) != i {
			var n int64
			for ; i < len(b) && b[i] >= '0' && b[i] <= '9'; i++ {
				n = int64(b[i]-'0') + n*10
			}
			if len(b) == i {
				if neg {
					n = -n
				}
				return n, nil
			}
		}
	}

	if n, err := strconv.ParseInt(string(b), 10, 64); err != nil {
		return 0, ErrorTrace(err)
	} else {
		return n, nil
	}
}

//3\r\nset\r\n
func (dec *Decoder) decodeBulkBytes() ([]byte, error) {
	len, err := dec.decodeInt()
	if err != nil {
		return nil, ErrorTrace(err)
	}
	switch {
	case len < -1:
		return nil, ErrorTrace(err)
	case len > MaxBulkBytesLen:
		return nil, ErrorTrace(err)
	case len == -1:
		return nil, nil
	}
	parts, err := dec.br.ReadFull(int(len) + 2)
	if err != nil {
		return nil, ErrorTrace(err)
	}
	if parts[len] != '\r' || parts[len+1] != '\n' {
		return nil, ErrorTrace(errorNew("decodeBulkBytes err"))
	}
	return parts[:len], nil
}

//*3\r\n$3\r\nset\r\n$3\r\nkey\r\n$5\r\nvalue\r\n
func (dec *Decoder) decodeArray() ([]*Resp, error) {
	len, err := dec.decodeInt()
	if err != nil {
		return nil, ErrorTrace(err)
	}
	switch {
	case len == -1:
		return nil, nil
	case len < -1:
		return nil, ErrorTrace(err)
	case len > MaxArrayLen:
		return nil, ErrorTrace(err)
	}
	array := make([]*Resp, len)
	for i := range array {
		r, err := dec.Decode()
		if err != nil {
			return nil, ErrorTrace(err)
		}
		array[i] = r
	}
	return array, nil
}

//在服务器处理客户端编码后产生的数据时调用此函数
//-----------------------1111111111111111111
func (dec *Decoder) DecodeMultiBulk() ([]*Resp, error) {
	if dec.Err != nil {
		return nil, ErrorTrace(errorNew("DecodeMultiBulk err"))
	}
	rep, err := dec.decodeMultiBulk()
	if err != nil {
		dec.Err = err
		log.Println("1111111")
	}
	return rep, dec.Err
}

//---------------------22222222222222222222222222
func (d *Decoder) decodeMultiBulk() ([]*Resp, error) {
	b, err := d.br.PeekByte()
	if err != nil {
		log.Println("2")
		return nil, ErrorTrace(err)
	}
	if byte(b) != TypeMultiReply {
		return d.decodeSingleLineMultiBulk() //-----------------333333333333333333333333333333
	}

	if _, err := d.br.ReadByte(); err != nil {
		log.Println("22")
		return nil, ErrorTrace(err)
	}
	n, err := d.decodeInt()

	if err != nil {
		log.Println("222")
		return nil, ErrorTrace(err)
	}
	switch {
	case n <= 0:
		return nil, ErrorTrace(ErrBadArrayLen)
	case n > MaxArrayLen:
		return nil, ErrorTrace(ErrBadArrayLenTooLong)
	}
	multi := make([]*Resp, n)
	for i := range multi {
		r, err := d.decodeResp()
		if err != nil {
			log.Println("2222")
			return nil, err
		}
		if r.Type != TypeBulkReply {
			log.Println("22222")
			return nil, ErrorTrace(ErrBadMultiBulkContent)
		}
		multi[i] = r
	}
	return multi, nil
}

//-----------------333333333333333333
func (d *Decoder) decodeSingleLineMultiBulk() ([]*Resp, error) {
	b, err := d.decodeTextBytes()
	if err != nil {
		log.Println("3333")
		return nil, err
	}
	multi := make([]*Resp, 0, 8)
	for l, r := 0, 0; r <= len(b); r++ {
		if r == len(b) || b[r] == ' ' {
			if l < r {
				multi = append(multi, NewBulkReply(b[l:r]))
			}
			l = r + 1
		}
	}
	if len(multi) == 0 {
		log.Println("33333")
		return nil, ErrorTrace(err)
	}
	return multi, nil
}

//回复类型
type Resp struct {
	Type byte

	Value []byte
	Array []*Resp
}

/*
状态回复（status reply）的第一个字节是 "+"
错误回复（error reply）的第一个字节是 "-"
整数回复（integer reply）的第一个字节是 ":"
批量回复（bulk reply）的第一个字节是 "$"
多条批量回复（multi bulk reply）的第一个字节是 "*"
*/

//------Response --------
const (
	TypeString     = '+'
	TypeError      = '-'
	TypeInt        = ':'
	TypeBulkReply  = '$'
	TypeMultiReply = '*'
)

func NewString(value []byte) (rep *Resp) {
	rep = &Resp{}
	rep.Type = TypeString
	rep.Value = value
	return rep

}
func NewInt(value []byte) (rep *Resp) {
	rep = &Resp{}
	rep.Type = TypeInt
	rep.Value = value
	return rep

}
func NewError(value []byte) (rep *Resp) {
	rep = &Resp{}
	rep.Type = TypeError
	rep.Value = value
	return rep

}
func NewBulkReply(value []byte) (rep *Resp) {
	rep = &Resp{}
	rep.Type = TypeBulkReply
	rep.Value = value
	return rep

}
func NewMultReply(arry []*Resp) (rep *Resp) {
	rep = &Resp{}
	rep.Type = TypeMultiReply
	rep.Array = arry
	return rep

}
func ErrorTrace(err error) error {

	if err != nil {
		log.Println("errors Tracing", err.Error())
	}
	return err
}

func ErrorNew(msg string) error {
	return errors.New("error occur,msg") // 将字符串 text 包装成一个 error 对象返回
}
