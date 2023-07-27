package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"strings"
)

var token string
var proxy string
var t string

func main() {
	flag.StringVar(&t, "t", "cmd", "模式，默认cmd模式。cmd:命令行模式 还是:server 模式")
	flag.StringVar(&proxy, "proxy", "HTTP_PROXY", "从环境变量读代理地址，默认HTTP_PROXY.")
	flag.StringVar(&token, "key", "", "openai key. 可以从这里获取：https://platform.openai.com/account/api-keys")
	flag.Parse()

	if token == "" {
		fmt.Println("必须带上api key. 可以从这里生成一个：https://platform.openai.com/account/api-keys")
		os.Exit(-1)
	}
	log.SetFlags(log.Ldate | log.Lshortfile)
	var err error
	globalClient, err = getClient(token, os.Getenv(proxy))
	if err != nil {
		log.Fatal(err)
	}
	hi, err := globalClient.unary(context.TODO(), "hi")
	if err != nil {
		panic(err)
	}
	fmt.Println(hi)

	args := os.Args
	if len(args) != 2 {
		fmt.Println("必须要一个参数", args)
		return
	}
	switch t {
	case "server":
		server()
	case "cmd":
		cmd()
	default:
		fmt.Println("只支持 server，cmd两种模式")
	}
}

var cmdEndSignal = "%%%"

func cmd() {
	for {
		reader := bufio.NewReader(os.Stdin)
		sb := strings.Builder{}
		for {
			str, err := reader.ReadString('\n')
			if err != nil {
				panic(err)
			}
			if strings.Contains(str, cmdEndSignal) {
				prefix := strings.TrimSuffix(str, cmdEndSignal)
				sb.WriteString(prefix)
				w := NewWriter(os.Stdout.Write, os.Stdout.Close, nil)
				err := globalClient.stream(context.TODO(), sb.String(), w)
				if err != nil {
					log.Println(err)
				}
				sb.Reset()
				continue
				//str = ss[1]
			}
			sb.WriteString(str)
		}
	}
}

func server() {
	r := gin.Default()
	r.GET("/v1/ask", streamAsk)
	log.Println(r.Run())
}

func streamAsk(ctx *gin.Context) {
	ctx.Header("Content-Type", "text/text; charset=utf-8")
	query, ok := ctx.GetQuery("query")
	if !ok {
		_, _ = ctx.Writer.Write([]byte("需要输入query"))
		return
	}
	w := NewWriter(ctx.Writer.Write, nil, ctx.Writer.Flush)
	err := globalClient.stream(ctx, query, w)
	log.Println(err)
}
