package pdf

import (
	"github.com/jellydator/ttlcache/v3"
)

type cache struct {
	c *ttlcache.Cache[string, []byte]
}

func (c cache) Get(key string) ([]byte, bool) {
	val := c.c.Get(key)
	return val.Value(), val == nil
}

func (c cache) Set(key string, val []byte) {
	c.c.Set(key, val, ttlcache.DefaultTTL)
}
