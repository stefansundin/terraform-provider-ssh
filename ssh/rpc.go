package ssh

import (
	"encoding/gob"
	"github.com/jinzhu/copier"
)

type SSHTunnelServer struct {
	Tunnel *SSHTunnel
	Ready  bool
}

func NewSSHTunnelServer(tunnel *SSHTunnel) *SSHTunnelServer {
	tunnelServer := &SSHTunnelServer{Tunnel: tunnel}
	gob.Register(SSHPrivateKey{})
	gob.Register(SSHAuthSock{})
	gob.Register(SSHPassword{})
	return tunnelServer
}

func (ts *SSHTunnelServer) GetSSHTunnel(ack *bool, tunnel *SSHTunnel) error {
	copier.Copy(tunnel, ts.Tunnel)
	return nil
}

func (ts *SSHTunnelServer) PutSSHReady(port int, ack *bool) error {
	ts.Tunnel.Local.Port = port
	ts.Ready = true
	return nil
}
