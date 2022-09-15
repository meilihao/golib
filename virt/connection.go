package virt

import (
	"errors"
	"fmt"
	"sync"

	"github.com/meilihao/golib/v2/log"
	"go.uber.org/zap"
	"libvirt.org/go/libvirt"
)

// from webvirtcloud vrtManager/connection.py
type ConnectionManager struct {
	KeepaliveInterval int
	KeepaliveCount    uint
	Connections       map[string]*VConnection
	ConnectionsLock   sync.RWMutex
	EventLoop         *ConnectionManagerEventLoop
}

type ConnectionManagerEventLoop struct {
}

type VConnection struct {
	Host     string
	Login    string // login username
	Password string
	Type     string
	Conn     *libvirt.Connect
}

func (el *ConnectionManagerEventLoop) Init() error {
	err := libvirt.EventRegisterDefaultImpl()
	if err != nil {
		log.Glog.Error("libvirtd EventRegisterDefaultImpl", zap.Error(err))
	}

	return err
}

func (el *ConnectionManagerEventLoop) Run() {
	for {
		if err := libvirt.EventRunDefaultImpl(); err != nil {
			log.Glog.Error("Listening to libvirt events failed, retrying.", zap.Error(err))
			//time.Sleep(time.Second)
		}

	}
}

func NewConnectionManager(keepaliveInterval int, keepaliveCount uint) (m *ConnectionManager, err error) {
	m = &ConnectionManager{
		KeepaliveInterval: keepaliveInterval,
		KeepaliveCount:    keepaliveCount,
		Connections:       map[string]*VConnection{},
		EventLoop:         new(ConnectionManagerEventLoop),
	}

	if err = m.EventLoop.Init(); err != nil {
		return
	}
	go m.EventLoop.Run()

	return
}

func (m *ConnectionManager) GetConnection(host, login, passwd, typ string) (conn *libvirt.Connect, err error) {
	var has bool
	var vconn *VConnection

	m.ConnectionsLock.RLock()
	vconn, has = m.Connections[host]
	if has {
		if isOk, _ := vconn.Conn.IsAlive(); isOk {
			conn = vconn.Conn
			m.ConnectionsLock.RUnlock()
			return
		}
		conn.UnregisterCloseCallback()
		conn.Close()
		delete(m.Connections, host)
	}
	m.ConnectionsLock.RUnlock()

	m.ConnectionsLock.Lock()
	defer m.ConnectionsLock.Unlock()

	vconn = &VConnection{
		Host:     host,
		Login:    login,
		Password: passwd,
		Type:     typ,
	}

	conn, err = vconn.NewConnect()
	if err != nil {
		log.Glog.Error("connect libvirtd", zap.Error(err))
		return
	}

	conn.SetKeepAlive(m.KeepaliveInterval, m.KeepaliveCount)
	conn.RegisterCloseCallback(m.CloseCallback)

	vconn.Conn = conn
	m.Connections[host] = vconn

	return
}

func (m *ConnectionManager) CloseCallback(conn *libvirt.Connect, reason libvirt.ConnectCloseReason) {
	// m.ConnectionsLock.Lock()
	// defer m.ConnectionsLock.Unlock()
}

const (
	LibvirtUriTypeSSh    = "ssh"
	LibvirtUriTypeTcp    = "tcp"
	LibvirtUriTypeTls    = "tls"
	LibvirtUriTypeSocket = "socket"
)

// driver[+transport]://[username@][hostname][:port]/[path][?extraparameters]
func (vc *VConnection) NewConnect() (conn *libvirt.Connect, err error) {
	var target string

	switch vc.Type {
	case LibvirtUriTypeTls:
		target = fmt.Sprintf("qemu+tls://%s@%s/system", vc.Login, vc.Host)
		auth := &libvirt.ConnectAuth{
			CredType: []libvirt.ConnectCredentialType{
				libvirt.CRED_AUTHNAME,
				libvirt.CRED_PASSPHRASE,
			},
			Callback: vc.LibvirtAuthCredentialsCallback,
		}
		conn, err = libvirt.NewConnectWithAuth(target, auth, 0)
	case LibvirtUriTypeTcp:
		target = fmt.Sprintf("qemu+tcp://%s/system", vc.Host)
		auth := &libvirt.ConnectAuth{
			CredType: []libvirt.ConnectCredentialType{
				libvirt.CRED_AUTHNAME,
				libvirt.CRED_PASSPHRASE,
			},
			Callback: vc.LibvirtAuthCredentialsCallback,
		}
		conn, err = libvirt.NewConnectWithAuth(target, auth, 0)
	case LibvirtUriTypeSSh:
		target = fmt.Sprintf("qemu+ssh://%s@%s/system", vc.Login, vc.Host)
		conn, err = libvirt.NewConnect(target)
	case LibvirtUriTypeSocket:
		target = "qemu:///system" //qemu+unix:///system?socket=/run/truenas_libvirt/libvirt-sock
		conn, err = libvirt.NewConnect(target)
	default:
		err = errors.New("invalid libvirt uri type")
	}

	return
}

func (vc *VConnection) LibvirtAuthCredentialsCallback(creds []*libvirt.ConnectCredential) {
	for _, cred := range creds {
		if cred.Type == libvirt.CRED_AUTHNAME {
			cred.Result = vc.Login
			if len(cred.Result) == 0 {
				cred.Result = cred.DefResult
			}
		} else if cred.Type == libvirt.CRED_PASSPHRASE {
			cred.Result = vc.Password
		}
	}
}
