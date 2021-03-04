// Package shared contains shared data between the host and plugins.
package shared

import (
	"context"

	"google.golang.org/grpc"

	"github.com/hashicorp/go-plugin"
	"github.com/stefansundin/terraform-provider-ssh/ssh/proto"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "tunnel",
}

var PluginMap = map[string]plugin.Plugin{
	"tunnel": &SSHTunnelGRPCPlugin{},
}

type SSHTunnel interface {
	Init(*proto.InitSSHTunnelRequest) (*proto.InitSSHTunnelResponse, error)
}

type SSHTunnelGRPCPlugin struct {
	plugin.Plugin
	Impl SSHTunnel
}

func (p *SSHTunnelGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterSSHTunnelServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *SSHTunnelGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewSSHTunnelClient(c)}, nil
}
