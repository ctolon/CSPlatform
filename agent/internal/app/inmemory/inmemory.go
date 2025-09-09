package inmemory

import (
	"sync"
)

var (
	usersCache sync.Map
	portsCache sync.Map
	once       sync.Once
)

type InMemoryCache struct{}

func NewInMemoryCache() *InMemoryCache {
	once.Do(func() {

		for _, u := range []string{"admin"} {
			usersCache.Store(u, struct{}{})
		}
		for _, p := range []int{80, 443, 3000, 5000, 8000, 8080, 8081, 8082, 9000} {
			portsCache.Store(p, struct{}{})
		}

	})
	return &InMemoryCache{}
}

func (c *InMemoryCache) IsValidPort(p int) bool {
	_, ok := portsCache.Load(p)
	return ok
}
