package main

import (
	"flag"
	"fmt"

	"github.com/wangzhou-ccc/beyond/application/follow/rpc/internal/config"
	"github.com/wangzhou-ccc/beyond/application/follow/rpc/internal/server"
	"github.com/wangzhou-ccc/beyond/application/follow/rpc/internal/svc"
	"github.com/wangzhou-ccc/beyond/application/follow/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/follow.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterFollowServer(grpcServer, server.NewFollowServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
