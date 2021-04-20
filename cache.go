package pdf

import (
	"github.com/ReneKroon/ttlcache/v2"
)

type cache struct {
	c *ttlcache.Cache
}

func (c cache) Get(key string) ([]byte, bool) {
	fontInterface, err := c.c.Get(key)
	if err != nil {
		return nil, false
	}

	fontBytes, ok := fontInterface.([]byte)
	return fontBytes, ok
}

func (c cache) Set(key string, val []byte) {
	c.c.Set(key, val)
}
