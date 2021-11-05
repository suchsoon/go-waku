package node

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"time"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/config"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	rendezvous "github.com/status-im/go-waku-rendezvous"
	"github.com/status-im/go-waku/waku/v2/protocol/store"
)

// Default clientId
const clientId string = "Go Waku v2 node"

type WakuNodeParameters struct {
	multiAddr      []ma.Multiaddr
	addressFactory basichost.AddrsFactory
	privKey        *crypto.PrivKey
	libP2POpts     []libp2p.Option

	enableRelay      bool
	enableFilter     bool
	isFilterFullNode bool
	wOpts            []pubsub.Option

	enableStore     bool
	shouldResume    bool
	storeMsgs       bool
	messageProvider store.MessageProvider
	maxMessages     int
	maxDays         time.Duration

	enableRendezvous       bool
	enableRendezvousServer bool
	rendevousStorage       rendezvous.Storage
	rendezvousOpts         []pubsub.DiscoverOpt

	keepAliveInterval time.Duration

	enableLightPush bool

	connStatusC chan ConnStatus
}

type WakuNodeOption func(*WakuNodeParameters) error

// MultiAddresses return the list of multiaddresses configured in the node
func (w WakuNodeParameters) MultiAddresses() []ma.Multiaddr {
	return w.multiAddr
}

// Identity returns a libp2p option containing the identity used by the node
func (w WakuNodeParameters) Identity() config.Option {
	return libp2p.Identity(*w.privKey)
}

// WithHostAddress is a WakuNodeOption that configures libp2p to listen on a list of net endpoint addresses
func WithHostAddress(hostAddr []*net.TCPAddr) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		var multiAddresses []ma.Multiaddr
		for _, addr := range hostAddr {
			hostAddrMA, err := manet.FromNetAddr(addr)
			if err != nil {
				return err
			}
			multiAddresses = append(multiAddresses, hostAddrMA)
		}

		params.multiAddr = append(params.multiAddr, multiAddresses...)

		return nil
	}
}

// WithAdvertiseAddress is a WakuNodeOption that allows overriding the addresses used in the waku node with custom values
func WithAdvertiseAddress(addressesToAdvertise []*net.TCPAddr, enableWS bool, wsPort int) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.addressFactory = func([]ma.Multiaddr) []ma.Multiaddr {
			var result []multiaddr.Multiaddr
			for _, adv := range addressesToAdvertise {
				addr, _ := manet.FromNetAddr(adv)
				result = append(result, addr)
				if enableWS {
					wsMa, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/ws", adv.IP.String(), wsPort))
					result = append(result, wsMa)
				}
			}
			return result
		}
		return nil
	}
}

// WithMultiaddress is a WakuNodeOption that configures libp2p to listen on a list of multiaddresses
func WithMultiaddress(addresses []ma.Multiaddr) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.multiAddr = append(params.multiAddr, addresses...)
		return nil
	}
}

// WithPrivateKey is used to set an ECDSA private key in a libp2p node
func WithPrivateKey(privKey *ecdsa.PrivateKey) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		privk := crypto.PrivKey((*crypto.Secp256k1PrivateKey)(privKey))
		params.privKey = &privk
		return nil
	}
}

// WithLibP2POptions is a WakuNodeOption used to configure the libp2p node.
// This can potentially override any libp2p config that was set with other
// WakuNodeOption
func WithLibP2POptions(opts ...libp2p.Option) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.libP2POpts = opts
		return nil
	}
}

// WithWakuRelay enables the Waku V2 Relay protocol. This WakuNodeOption
// accepts a list of WakuRelay gossipsub option to setup the protocol
func WithWakuRelay(opts ...pubsub.Option) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRelay = true
		params.wOpts = opts
		return nil
	}
}

// WithRendezvous is a WakuOption used to enable go-waku-rendezvous discovery.
// It accepts an optional list of DiscoveryOpt options
func WithRendezvous(discoverOpts ...pubsub.DiscoverOpt) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRendezvous = true
		params.rendezvousOpts = discoverOpts
		return nil
	}
}

// WithRendezvousServer is a WakuOption used to set the node as a rendezvous
// point, using an specific storage for the peer information
func WithRendezvousServer(storage rendezvous.Storage) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRendezvousServer = true
		params.rendevousStorage = storage
		return nil
	}
}

// WithWakuFilter enables the Waku V2 Filter protocol. This WakuNodeOption
// accepts a list of WakuFilter gossipsub options to setup the protocol
func WithWakuFilter(fullNode bool) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableFilter = true
		params.isFilterFullNode = fullNode
		return nil
	}
}

// WithWakuStore enables the Waku V2 Store protocol and if the messages should
// be stored or not in a message provider
func WithWakuStore(shouldStoreMessages bool, shouldResume bool) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableStore = true
		params.storeMsgs = shouldStoreMessages
		params.shouldResume = shouldResume
		return nil
	}
}

// WithWakuStoreAndLimits enables the Waku V2 Store protocol, storing them in an optional message provider
// applying an specific retention policy
func WithWakuStoreAndLimits(shouldResume bool, maxDays time.Duration, maxMessages int) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableStore = true
		params.storeMsgs = true
		params.shouldResume = shouldResume
		params.maxDays = maxDays
		params.maxMessages = maxMessages
		return nil
	}
}

// WithMessageProvider is a WakuNodeOption that sets the MessageProvider
// used to store and retrieve persisted messages
func WithMessageProvider(s store.MessageProvider) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.messageProvider = s
		return nil
	}
}

// WithLightPush is a WakuNodeOption that enables the lightpush protocol
func WithLightPush() WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableLightPush = true
		return nil
	}
}

// WithKeepAlive is a WakuNodeOption used to set the interval of time when
// each peer will be ping to keep the TCP connection alive
func WithKeepAlive(t time.Duration) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.keepAliveInterval = t
		return nil
	}
}

// WithConnectionStatusChannel is a WakuNodeOption used to set a channel where the
// connection status changes will be pushed to. It's useful to identify when peer
// connections and disconnections occur
func WithConnectionStatusChannel(connStatus chan ConnStatus) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.connStatusC = connStatus
		return nil
	}
}

// Default options used in the libp2p node
var DefaultLibP2POptions = []libp2p.Option{
	libp2p.DefaultTransports,
	libp2p.UserAgent(clientId),
	libp2p.NATPortMap(),       // Attempt to open ports using uPNP for NATed hosts.
	libp2p.EnableNATService(), // TODO: is this needed?)
	libp2p.ConnectionManager(connmgr.NewConnManager(200, 300, 0)),
}
