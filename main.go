package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/DeanThompson/ginpprof"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/configor"
	_ "github.com/mattn/go-oci8"
	"gopkg.in/olahol/melody.v1"
)

const SIZE = 65535 //max message size

type Config struct {
	AppName string `default:"WeeklyReport"`

	Server struct {
		Ip   string `default:"0.0.0.0"`
		Port string `default:"5000"`
	}

	Debug bool `default:"false"`

	DB struct {
		User     string `required:"true"`
		Password string `required:"true"`
		TnsName  string `required:"true"`
		Table    string `default:"weekly"`
	}
}

func main() {
	Config := Config{}
	if err := configor.Load(&Config, "config.yaml"); err != nil {
		log.Fatalln(err)
	}

	connect := Config.DB.User + "/" + Config.DB.Password + "@" + Config.DB.TnsName
	db, err := sql.Open("oci8", connect)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Query(fmt.Sprintf("select count(1) from %v", Config.DB.Table))
	if err != nil {
		log.Fatalln("数据库连接失败!", err)
	} else {
		log.Println("数据库连接成功!")
	}

	if !Config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	m := melody.New()

	ginpprof.Wrap(r)

	m.Upgrader = &websocket.Upgrader{
		ReadBufferSize:  SIZE,
		WriteBufferSize: SIZE,
	}
	m.Config.MaxMessageSize = int64(SIZE)
	m.Config.MessageBufferSize = 2048

	r.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	r.GET("/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		success_count := 0
		fail_count := 0
		t := time.Now()

		log.Printf("%v 收到数据：\n %v\n", t, string(msg))

		rows := GetRows(string(msg))
		rows_length := len(rows)

		var result string
		var wg = &sync.WaitGroup{}
		//var ok chan int

		for _, lines := range rows {
			//if matched, _ := regexp.MatchString(`^\^.*\$$`, lines); matched {
			row := strings.Replace(strings.Replace(lines, "^", "", 1), "$", "", 1) //去除开头"^"符号,去除结尾"$"符号
			row = strings.TrimSpace(strings.Replace(row, "\n", "", -1))            //去除换行符&去除空格
			key := strings.Split(row, "/")
			if len(key) != 6 {
				result += fmt.Sprintf("%v <span style='color: red;'>格式错误：</span>%v\n", t, lines)
				fail_count += 1
			} else {
				wg.Add(1)
				go func() {
					_, err := db.Exec(fmt.Sprintf("insert into %v(proj_name, proj_code, worker,work_content, problem, next_plan) values(:1,:2,:3,:4,:5,:6)", Config.DB.Table), key[0], key[1], key[2], key[3], key[4], key[5])
					if err != nil {
						log.Println(err)
					}
					wg.Done()
				}()
				success_count += 1
			}
			//}
		}
		result += fmt.Sprintf("%v 共提交 <span style='color: blue;'>%d</span> 条,成功 <span style='color: green;'>%d</span> 条,错误 <span style='color: red;'>%d</span> 条!\n", t, rows_length, success_count, fail_count)
		log.Println(result)
		//		if err = m.Broadcast([]byte(result)); err != nil {
		//			log.Println(err)
		//		}
		var session []*melody.Session
		session = append(session, s)
		if err = m.BroadcastMultiple([]byte(result), session); err != nil {
			log.Println(err)
		}
	})

	r.Run(Config.Server.Ip + ":" + Config.Server.Port)
}

func GetRows(content string) (rows []string) {
	for _, row := range strings.Split(content, "\n") {
		if len(row) != 0 {
			rows = append(rows, row)
		}
	}
	return
}
