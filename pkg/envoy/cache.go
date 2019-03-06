package envoy

import (
	"context"
	"sync"

	"github.com/ansel1/merry"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/rs/zerolog"
)

type Cache struct {
	envoycache.SnapshotCache

	mutex       sync.RWMutex
	lastVersion string
}

func NewCache(ctx context.Context) *Cache {
	logger := zerolog.Ctx(ctx)

	return &Cache{
		SnapshotCache: envoycache.NewSnapshotCache(true, NodeHash{}, NewLogger(logger)),
	}
}

func (c *Cache) UpdateSnapshot(node string, version string, snapshot envoycache.Snapshot) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.SnapshotCache.SetSnapshot(node, snapshot); err != nil {
		return merry.Wrap(err)
	}

	c.lastVersion = version
	return nil
}

func (c *Cache) ShouldUpdate(version string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.lastVersion != version
}
