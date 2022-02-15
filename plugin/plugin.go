package plugin

import (
	context "context"
	"fmt"
	"os/exec"

	wrapping "github.com/hashicorp/go-kms-wrapping/v2"
	"github.com/hashicorp/go-plugin"
	gp "github.com/hashicorp/go-plugin"
	grpc "google.golang.org/grpc"
)

// HandshakeConfig is a shared config that can be used regardless of wrapper, to
// avoid having to know type-specific things about each plugin
var HandshakeConfig = gp.HandshakeConfig{
	MagicCookieKey:   "HASHICORP_GKW_PLUGIN",
	MagicCookieValue: "wrapper",
}

// wrapper embeds Plugin and is used as the top-level
type wrapper struct {
	gp.Plugin

	impl wrapping.Wrapper
}

// ServePlugin is a generic function to start serving a wrapper as a plugin
func ServePlugin(wrapper wrapping.Wrapper, opt ...Option) error {
	opts, err := getOpts(opt...)
	if err != nil {
		return err
	}
	wrapServer, err := NewWrapperServer(wrapper)
	if err != nil {
		return err
	}
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		VersionedPlugins: map[int]plugin.PluginSet{
			1: {"wrapping": wrapServer},
		},
		Logger:     opts.withLogger,
		GRPCServer: plugin.DefaultGRPCServer,
	})
	return nil
}

func NewWrapperServer(impl wrapping.Wrapper) (*wrapper, error) {
	if impl == nil {
		return nil, fmt.Errorf("empty underlying wrapper passed in")
	}

	return &wrapper{
		impl: impl,
	}, nil
}

func NewWrapperClient(pluginPath string, opt ...Option) (*gp.Client, error) {
	opts, err := getOpts(opt...)
	if err != nil {
		return nil, err
	}

	return gp.NewClient(&gp.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		VersionedPlugins: map[int]gp.PluginSet{
			1: {"wrapping": &wrapper{}},
		},
		Cmd: exec.Command(pluginPath),
		AllowedProtocols: []gp.Protocol{
			gp.ProtocolGRPC,
		},
		Logger:   opts.withLogger,
		AutoMTLS: true,
	}), nil
}

func (w *wrapper) GRPCServer(broker *gp.GRPCBroker, s *grpc.Server) error {
	RegisterWrappingServer(s, &wrapServer{impl: w.impl})
	if initFinalizer, ok := w.impl.(wrapping.InitFinalizer); ok {
		RegisterInitFinalizeServer(s, &initFinalizeServer{impl: initFinalizer})
	}
	if hmacComputer, ok := w.impl.(wrapping.HmacComputer); ok {
		RegisterHmacComputerServer(s, &hmacComputerServer{impl: hmacComputer})
	}
	return nil
}

func (w *wrapper) GRPCClient(ctx context.Context, broker *gp.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	wrap := &wrapClient{impl: NewWrappingClient(c)}
	switch {
	case w.hmacComputer && w.initFinalizer:
		ifc := &wrapInitFinalizerClient{
			wrapClient: wrap,
			impl:       NewInitFinalizeClient(c),
		}
		return &wrapInitFinalizerHmacComputerClient{
			wrapInitFinalizerClient: ifc,
			impl:                    NewHmacComputerClient(c),
		}, nil
	case w.initFinalizer:
		return &wrapInitFinalizerClient{
			wrapClient: wrap,
			impl:       NewInitFinalizeClient(c),
		}, nil
	default:
		return wrap, nil
	}
}
