package xclient

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type GeeRegistryDiscovery struct {
	*MultiServerDiscovery
	registry   string        // registry center address
	timeout    time.Duration // registry center expiration time
	lastUpdate time.Time
}

const defaultUpdateTimeout = time.Second * 10

func NewGeeRegistryDiscovery(registryAddr string, timeout time.Duration) *GeeRegistryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}

	return &GeeRegistryDiscovery{
		MultiServerDiscovery: NewMultiServerDiscovery(make([]string, 0)),
		registry:             registryAddr,
		timeout:              timeout,
	}
}

var _ Discovery = (*GeeRegistryDiscovery)(nil)

func (discovery *GeeRegistryDiscovery) Update(servers []string) error {
	discovery.mu.Lock()
	defer discovery.mu.Unlock()

	discovery.servers = servers
	discovery.lastUpdate = time.Now()
	return nil
}

func (discovery *GeeRegistryDiscovery) Refresh() error {
	discovery.mu.Lock()
	defer discovery.mu.Unlock()

	// don't need to update
	if discovery.lastUpdate.Add(discovery.timeout).After(time.Now()) {
		return nil
	}

	// need to update
	log.Println("rpc registry: refresh servers from registry", discovery.registry)
	res, err := http.Get(discovery.registry)
	if err != nil {
		log.Println("rpc registry refresh err:", err)
		return err
	}

	servers := strings.Split(res.Header.Get("X-Geerpc-Servers"), ",")
	discovery.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		s := strings.TrimSpace(server)
		if s != "" {
			discovery.servers = append(discovery.servers, s)
		}
	}
	discovery.lastUpdate = time.Now()

	return nil
}

func (discovery *GeeRegistryDiscovery) Get(mode SelectMode) (string, error) {
	// check whether the registry center is out of date
	if err := discovery.Refresh(); err != nil {
		return "", err
	}
	return discovery.MultiServerDiscovery.Get(mode)
}

func (discovery *GeeRegistryDiscovery) GetAll() ([]string, error) {
	if err := discovery.Refresh(); err != nil {
		return nil, err
	}
	return discovery.MultiServerDiscovery.GetAll()
}
