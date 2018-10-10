package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"syscall"
)

func AppendToFile(fileName string, cbuf string) error {
	//一直写的方式打开文件
	f, err := os.OpenFile(fileName, os.O_WRONLY|syscall.O_CREAT, 0644)
	if err != nil {
		log.Println("log file open failed" + err.Error())
	} else {
		n, _ := f.Seek(0, os.SEEK_END)
		_, err = f.WriteAt([]byte(cbuf), n)

	}
	defer f.Close()
	return err
}
func ReadAof(filename string) []string {
	//首先判断该文件是否存在
	if _, err := os.Stat(filename); err != nil {
		//如果不存在，创建一个文件然后返回
		if os.IsNotExist(err) {
			_, err := os.Create(filename)
			if err != nil {
				log.Println("ReadAof Create file failed" + err.Error())
			}
		}
		return nil
	} else {

		//首先根据文件名打开文件
		file, err := os.Open(filename)
		//然后根据俄每一个\对命令进行分割
		if err != nil {
			log.Println("ReadAof failed")

		}
		defer file.Close()
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Println("aof file read failed" + err.Error())
		}
		ret := bytes.Split(content, []byte{'*'}) //????????????????
		var pos = make([]string, len(ret))
		for key, val := range ret[1:] {
			val := append(val[:0], append([]byte{'*'}, val[0:]...)...)
			pos[key] = string(val)
		}
		return pos
	}
}
