package cache

import (
	"sync"
)

type Cache struct {
	mu sync.RWMutex
	data map[string]string
}

func (c *Cache) Read(key string) (string,bool) {
	c.mu.RLock()
	 defer c.mu.RUnlock()
	 val,ok := c.data[key]
	 return val,ok
}

func (c *Cache) Upsert(key string,value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key]=value
}

func New() *Cache {
	var c Cache
	c.data = make(map[string]string)
	return &c
}


