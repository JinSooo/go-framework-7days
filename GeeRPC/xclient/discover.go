package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

/* ---------------------------------- 服务发现 ---------------------------------- */

type SelectMode int

// 负载均衡策略
const (
	RandomSelect = iota
	RoundRobinSelect
)

// 服务发现
type Discovery interface {
	// refresh from remote registry
	Refresh() error
	Update(servers []string) error
	// select a server by mode
	Get(mode SelectMode) (string, error)
	GetAll() ([]string, error)
}

/* ------------------------- 不需要注册中心，服务列表由手工维护的服务发现 ------------------------- */

type MultiServerDiscovery struct {
	r       *rand.Rand // use for RandomSelect
	index   int        // use for RoundRobinSelect
	mu      sync.RWMutex
	servers []string
}

func NewMultiServerDiscovery(servers []string) *MultiServerDiscovery {
	discovery := &MultiServerDiscovery{
		// generate a random seed
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
		servers: servers,
	}
	// generate a random
	discovery.index = discovery.r.Intn(math.MaxInt32 - 1)
	return discovery
}

// implement Discovery interface
var _ Discovery = (*MultiServerDiscovery)(nil)

// Refresh doesn't make sens for MultiServerDiscovery, so ignore it
func (discovery *MultiServerDiscovery) Refresh() error {
	return nil
}

func (discovery *MultiServerDiscovery) Update(servers []string) error {
	discovery.mu.Lock()
	defer discovery.mu.Unlock()

	discovery.servers = servers
	return nil
}

func (discovery *MultiServerDiscovery) Get(mode SelectMode) (string, error) {
	discovery.mu.Lock()
	defer discovery.mu.Unlock()

	n := len(discovery.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}

	switch mode {
	case RandomSelect:
		return discovery.servers[discovery.r.Intn(n)], nil
	case RoundRobinSelect:
		server := discovery.servers[discovery.index%n]
		discovery.index = (discovery.index + 1) % n
		return server, nil
	default:
		return "", errors.New("rpc discovery: not supported select mode")
	}
}

func (discovery *MultiServerDiscovery) GetAll() ([]string, error) {
	discovery.mu.Lock()
	defer discovery.mu.Unlock()

	servers := make([]string, len(discovery.servers))
	copy(servers, discovery.servers)

	return servers, nil
}
