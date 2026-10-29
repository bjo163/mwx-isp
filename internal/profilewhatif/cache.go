package profilewhatif

import (
	"errors"

	"github.com/bjo163/mwx-isp/internal/domain"
)

// cacheView is the whole contract the study hands the product's inheritance
// getters: one read method, matching domain.ProfileCacheGetter exactly. It has
// no write method, no eviction method and no database behind it, which is what
// "read-only" means here.
type cacheView interface {
	Get(profileID int64) (*domain.RadiusProfile, error)
}

// readOnlyCache serves fixture profiles and counts every lookups the product's
// own getters make, so the cost of one Access-Accept is a measured number.
type readOnlyCache struct {
	profiles map[int64]*domain.RadiusProfile
	gets     int
}

func newReadOnlyCache(profiles map[int64]*domain.RadiusProfile) *readOnlyCache {
	return &readOnlyCache{profiles: profiles}
}

// Get is the only method on the type; a test asserts that with reflection.
func (c *readOnlyCache) Get(profileID int64) (*domain.RadiusProfile, error) {
	c.gets++
	p, ok := c.profiles[profileID]
	if !ok || p == nil {
		return nil, errProfileMissing
	}
	return p, nil
}

// errProfileMissing stands in for the database error the product's cache returns
// when the linked profile row cannot be read.
var errProfileMissing = errors.New("whatif: profile row is not in the cache")

// brokenCache always fails, which is how the study asks what happens when the
// linked profile cannot be resolved at authentication time.
type brokenCache struct{}

// Get always returns an error.
func (brokenCache) Get(int64) (*domain.RadiusProfile, error) {
	return nil, errProfileMissing
}

// lookupsOf reads the counter out of a cache view.
func lookupsOf(v cacheView) int {
	if rc, ok := v.(*readOnlyCache); ok {
		return rc.gets
	}
	return 0
}
