package grpcx

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"time"
	"zkit/logger"
	"zkit/netx"
)

type Server struct {
	*grpc.Server
	Addr        string
	Port        int
	EtcdAddr    string
	EtcdTTL     int64
	etcdClient  *clientv3.Client
	etcdManager endpoints.Manager
	etcdKey     string
	cancel      func()
	Name        string
	L           logger.LoggerV1
}

// Serve 启动服务器并且阻塞
func (s *Server) Serve() error {
	// 初始化一个控制整个过程的 ctx
	// 你也可以考虑让外面传进来，这样的话就是 main 函数自己去控制了
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	port := strconv.Itoa(s.Port)
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}
	// 要先确保启动成功，再注册服务
	err = s.register(ctx, port)
	if err != nil {
		return err
	}
	return s.Server.Serve(l)
}

func (s *Server) register(ctx context.Context, port string) error {
	cli, err := clientv3.NewFromURL("http://localhost:12379")
	if err != nil {
		return err
	}
	s.etcdClient = cli
	serviceName := "service/" + s.Name
	em, err := endpoints.NewManager(cli,
		serviceName)
	if err != nil {
		return err
	}
	s.etcdManager = em
	ip := netx.GetOutboundIP()
	s.etcdKey = serviceName + "/" + ip
	addr := ip + ":" + port
	leaseResp, err := cli.Grant(ctx, s.EtcdTTL)
	// 开启续约
	ch, err := cli.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		return err
	}
	go func() {
		// 可以预期，当我们的 cancel 被调用的时候，就会退出这个循环
		for chResp := range ch {
			s.L.Debug("续约：", logger.String("resp", chResp.String()))
		}
	}()
	// metadata 我们这里没啥要提供的
	return em.AddEndpoint(ctx, s.etcdKey,
		endpoints.Endpoint{Addr: addr}, clientv3.WithLease(leaseResp.ID))
}

func (s *Server) Close() error {
	s.cancel()
	if s.etcdManager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err := s.etcdManager.DeleteEndpoint(ctx, s.etcdKey)
		if err != nil {
			return err
		}
	}
	err := s.etcdClient.Close()
	if err != nil {
		return err
	}
	s.Server.GracefulStop()
	return nil
}
