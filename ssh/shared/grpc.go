package shared

import (
	"github.com/stefansundin/terraform-provider-ssh/ssh/proto"
	"golang.org/x/net/context"
)

type GRPCClient struct{ client proto.SSHTunnelClient }

func (m *GRPCClient) Init(request *proto.InitSSHTunnelRequest) (*proto.InitSSHTunnelResponse, error) {
	return m.client.Init(context.Background(), request)
}

type GRPCServer struct {
	Impl SSHTunnel
}

func (m *GRPCServer) Init(
	ctx context.Context,
	req *proto.InitSSHTunnelRequest) (*proto.InitSSHTunnelResponse, error) {
	return m.Impl.Init(req)
}
