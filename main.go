package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/marsli9945/go-websocket/impl"
	"io/ioutil"
	"log"
	"net/http"
	"time"
	//"time"
)

var (
	upgrader = websocket.Upgrader{
		// 读取存储空间大小
		ReadBufferSize: 1024,
		// 写入存储空间大小
		WriteBufferSize: 1024,
		// 允许跨域
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	// 在线用户和链接凭据
	userList = map[string]*impl.Connection{}
	// 请求参数
	param struct {
		Name string      `json:"name"`
		Data interface{} `json:"data"`
	}
)

// http返回参数
type result struct {
	Code int
	Msg  string
	data interface{}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	var (
		wsConn *websocket.Conn
		err    error
		// data []byte
		conn *impl.Connection
		data []byte
	)
	// 完成http应答，在httpheader中放下如下参数
	if wsConn, err = upgrader.Upgrade(w, r, nil); err != nil {
		return // 获取连接失败直接返回
	}

	if conn, err = impl.InitConnection(wsConn); err != nil {
		goto ERR
	}

	go func() {
		var (
			err error
		)
		for {
			// 每隔一秒发送一次心跳
			if err = conn.WriteMessage([]byte("heartbeat")); err != nil {
				return
			}
			time.Sleep(1 * time.Second)
		}

	}()

	for {
		if data, err = conn.ReadMessage(); err != nil {
			goto ERR
		}

		if err := json.Unmarshal(data, &param); err != nil {
			log.Println(err)
		}

		userList[param.Name] = conn

		if err = conn.WriteMessage(data); err != nil {
			goto ERR
		}
	}

ERR:
	// 关闭当前连接
	conn.Close()
}

func main() {
	// 当有请求访问ws时，执行此回调方法
	http.HandleFunc("/websocket", wsHandler)

	// 消息推动的对外http接口
	http.HandleFunc("/websocket/push", func(writer http.ResponseWriter, request *http.Request) {
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			log.Println(err)
		}
		err = json.Unmarshal(body, &param)
		if err != nil {
			log.Println(err)
		}

		var r []byte

		if param.Name == "" {
			r, _ = json.Marshal(&result{401, "请使用name参数指定接收人", nil})
		} else {
			if v, ok := userList[param.Name]; !ok || v.IsClosed {
				delete(userList, param.Name) // 清理断开的连接
				log.Println(param.Name + "------未上线")
				r, _ = json.Marshal(&result{401, "用户已断开链接", nil})
			} else {
				err = userList[param.Name].WriteMessage(body)
				if err != nil {
					log.Println(err)
				}
				log.Println(param.Name + "+++++++发送成功")
				r, _ = json.Marshal(&result{200, "推送成功", nil})
			}
		}

		_, _ = writer.Write(r)
	})

	// 获取在线的用户列表
	http.HandleFunc("/websocket/list", func(writer http.ResponseWriter, request *http.Request) {
		var users []string
		for k, v := range userList {
			if !v.IsClosed {
				users = append(users, k)
			} else {
				delete(userList, k) //清理断开的连接
			}
		}
		list, _ := json.Marshal(users)
		_, _ = writer.Write(list)
	})

	// 渲染html文件进行测试
	http.HandleFunc("/websocket/test", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, "html/index.html")
	})

	// 监听127.0.0.1:7777
	err := http.ListenAndServe("0.0.0.0:7777", nil)
	if err != nil {
		log.Fatal("ListenAndServe", err.Error())
	}
}
