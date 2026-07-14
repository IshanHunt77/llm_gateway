package cache

import "testing"

func TestCache(t *testing.T) {
	c := New()

	t.Run("write six", func(t *testing.T) {
		t.Parallel()
		c.Upsert("6", "six")
	})

	t.Run("write kuus", func(t *testing.T) {
		t.Parallel()
		c.Upsert("6", "kuus")
	})
}