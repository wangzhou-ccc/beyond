package main

import (
	"flag"
	"fmt"

	"github.com/wangzhou-ccc/beyond/application/applet/internal/config"
	"github.com/wangzhou-ccc/beyond/application/applet/internal/handler"
	"github.com/wangzhou-ccc/beyond/application/applet/internal/svc"
	"github.com/wangzhou-ccc/beyond/pkg/xcode"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/applet-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// 自定义错误处理方法
	httpx.SetErrorHandler(xcode.ErrHandler)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
