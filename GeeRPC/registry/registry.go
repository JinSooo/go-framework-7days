package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

/* ---------------------------------- 注册中心 ---------------------------------- */
/**
 * 注册中心的好处在于，客户端和服务端都只需要感知注册中心的存在，而无需感知对方的存在。
 * 	服务端启动后，向注册中心发送注册消息，注册中心得知该服务已经启动，处于可用状态。一般来说，服务端还需要定期向注册中心发送心跳，证明自己还活着。
 * 	客户端向注册中心询问，当前哪天服务是可用的，注册中心将可用的服务列表返回客户端。
 * 	客户端根据注册中心得到的服务列表，选择其中一个发起调用。
 */

// Registry Center
type GeeRegistry struct {
	mutex   sync.Mutex
	timeout time.Duration          // a server expiration time
	servers map[string]*ServerItem // registered servers
}

// a registered server Info
type ServerItem struct {
	Addr  string
	start time.Time // register time
}

const (
	defaultPath    = "/_geerpc/registry"
	defaultTimeout = time.Minute * 5
)

func NewRegistry(timeout time.Duration) *GeeRegistry {
	return &GeeRegistry{
		timeout: timeout,
		servers: make(map[string]*ServerItem),
	}
}

var DefaultRegistry = NewRegistry(defaultTimeout)

// register server, if server existed, server's start will be updated
func (registry *GeeRegistry) putServer(addr string) {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	server := registry.servers[addr]
	if server == nil {
		registry.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	} else {
		server.start = time.Now()
	}
}

func (registry *GeeRegistry) aliveServer() []string {
	registry.mutex.Lock()
	defer registry.mutex.Unlock()

	var alive []string
	for addr, server := range registry.servers {
		if registry.timeout == 0 || server.start.Add(registry.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else {
			delete(registry.servers, addr)
		}
	}

	sort.Strings(alive)
	return alive
}

/**
 * HTTP
 * 	Get：返回所有可用的服务列表，通过自定义字段 X-Geerpc-Servers 承载。
 * 	Post：添加服务实例或发送心跳，通过自定义字段 X-Geerpc-Server 承载。
 */

func (registry *GeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		w.Header().Set("X-Geerpc-Servers", strings.Join(registry.aliveServer(), ","))
	case "POST":
		addr := req.Header.Get("X-Geerpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		registry.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (registry *GeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, registry)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultRegistry.HandleHTTP(defaultPath)
}

/**
 * Heartbeat
 *	Heartbeat send a heartbeat message every once in a while
 * notice: This should be used by the server
 */
func Heartbeat(registry string, addr string, duration time.Duration) {
	if duration == 0 {
		// make sure there is enough time to send heart beat, before it's removed from registry
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}

	var err error
	err = sendHeartbeat(registry, addr)

	go func() {
		// set a timer to send heartbeat every duration time
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry string, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Geerpc-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
