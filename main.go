package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tarm/goserial"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type conf struct {
	Com string `yaml:"com"`
	Url string `yaml:"url"`
}

func (c *conf) read() *conf {
	yamlFile, err := ioutil.ReadFile("conf.yml")
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatal(err)
	}
	return c
}

type request struct {
	Token  string `json:"token"`
	Url    string `json:"url"`
	Method string `json:"method"`
	Code   string `json:"code"`
	Body   []byte `json:"body"`
}

func main() {
	var conf conf
	conf.read()
	b, _ := json.Marshal(&conf)
	fmt.Println(string(b))

	myHandler := &MyHandler{conf.Url}

	c := &serial.Config{Name: conf.Com, Baud: 9600}
	//打开串口
	s, err := serial.OpenPort(c)
	defer s.Close()
	if err != nil {
		log.Fatal(err)
	}

	for {
		req, err := ReadRequest(s) //读取请求
		if err != nil {
			log.Println(err)
			continue
		}
		err = myHandler.Process(s, req) //发送请求并响应
		if err != nil {
			log.Println(err)
			s.Write([]byte(err.Error()))
			s.Write([]byte("#end#"))
		}
	}

}

type MyHandler struct {
	baseUrl string
}

func ReadRequest(s io.ReadWriteCloser) (*request, error) {
	data := make([][]byte, 0)
	buf := make([]byte, 1024)
	end := []byte("#end#")
	for {
		n, err := s.Read(buf)
		if err != nil && err != io.EOF {
			log.Println(err)
			return nil, err
		}
		if n > 0 {
			tmp := make([]byte, n)
			copy(tmp, buf[0:n])
			data = append(data, tmp)
		}
		data1 := bytes.Join(data, make([]byte, 0))
		if bytes.HasSuffix(data1, end) { //判断是否请求读取结束
			data2 := data1[0 : len(data1)-len(end)]
			fmt.Println(string(data2))
			request := &request{}
			err := json.Unmarshal(data2, request)
			return request, err
		}
	}
}

func (h *MyHandler) Process(s io.ReadWriteCloser, request *request) error {
	url := h.baseUrl + request.Url
	fmt.Println(url)
	r, e := http.NewRequest(request.Method, url, bytes.NewReader(request.Body))
	if e != nil {
		return e
	}
	r.Header.Set("Authorization", request.Token)
	r.Header.Set("X-Bank-Code", request.Code)
	r.Header.Set("Content-Type", "application/json;charset=UTF-8")
	resp, e1 := http.DefaultClient.Do(r)
	if e1 != nil {
		return e1
	}
	//开始写响应
	buf := make([]byte, 1024)
	end := []byte("#end#")
	data := make([][]byte, 0)
	data = append(data, []byte(request.Url + "\n"))
	for {
		n, e2 := resp.Body.Read(buf)
		if e2 != nil && e2 != io.EOF {
			return e2
		}
		if n > 0 {
			tmp := make([]byte, n)
			copy(tmp, buf)
			data = append(data, tmp)
			//_, e3 := s.Write(buf[0:n])
			//if e3 != nil {
			//	return e3
			//}
		} else { //结束了
			//_, e3 := s.Write(end)
			//fmt.Println(m)
			//return e3
			break
		}
	}
	data = append(data, end)
	dataByte := bytes.Join(data, make([]byte, 0))
	fmt.Println(string(dataByte))

	fmt.Println(time.Now())
	index := 0
	count := 0
	delta := 1024
	for {
		index2 := index + delta
		if len(dataByte) - count < delta {
			index2 = len(dataByte)
		}
		_, e3 := s.Write(dataByte[index:index2])
		if e3 != nil {
			return e3
		}
		time.Sleep(time.Millisecond * 800)
		count += index2 - index
		index = index2
		if index2 == len(dataByte) {
			break
		}
	}

	fmt.Println(time.Now())

	fmt.Println(len(dataByte))
	return nil
}
