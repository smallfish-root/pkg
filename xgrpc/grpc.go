package xgrpc

import (
	"context"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"net"
	"sync"
	"time"
)

var (
	once     sync.Once
	grpcConn = make(map[string]*grpc.ClientConn)
)

type GrpcClientConf struct {
	Alias string
	Addr  string
}

func newGrpcClient(addr string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	retryOps := []grpc_retry.CallOption{
		grpc_retry.WithMax(3),
		grpc_retry.WithPerRetryTimeout(time.Second * 2),
		grpc_retry.WithBackoff(grpc_retry.BackoffLinearWithJitter(time.Second/2, 0.2)),
	}
	retry := grpc_retry.UnaryClientInterceptor(retryOps...)
	// lb: k8s headless svc(dns:///)
	opts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(retry),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             time.Second,
			PermitWithoutStream: true,
		}),
	}
	c, err := grpc.DialContext(ctx, fmt.Sprintf("dns:///%s", addr), opts...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return c, nil
}

func NewGrpcClient(conf []*GrpcClientConf) []*grpc.ClientConn {
	clients := make([]*grpc.ClientConn, 0, len(conf))
	once.Do(func() {
		for _, c := range conf {
			cli, err := newGrpcClient(c.Addr)
			if err != nil {
				panic(err)
			}
			grpcConn[c.Alias] = cli
			clients = append(clients, cli)
		}
	})
	return clients
}

func GetGrpcClient(alias string) *grpc.ClientConn {
	return grpcConn[alias]
}

type GrpcServerConf struct {
	NetWork string
	Addr    string
}

type Server struct {
	Obj      interface{}
	Register func(s *grpc.Server, obj interface{})
}

func (s *Server) NewGrpcServer(conf *GrpcServerConf) (*grpc.Server, error) {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(),
		)), grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Second,
			MaxConnectionAge:      30 * time.Second,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  5 * time.Second,
			Timeout:               1 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	srv := grpc.NewServer(opts...)
	s.Register(srv, s.Obj)
	listen, err := net.Listen(conf.NetWork, conf.Addr)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	go func() {
		err = srv.Serve(listen)
		if err != nil {
			panic(errors.WithStack(err))
		}
	}()
	return srv, nil
}
