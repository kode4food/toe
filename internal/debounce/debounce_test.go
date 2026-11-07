package debounce_test

import (
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/toe/internal/debounce"
)

const (
	debounceDelay = 50 * time.Millisecond
	debounceBurst = 100
)

func TestKeyed(t *testing.T) {
	t.Run("runs only the last call", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			k := debounce.NewKeyed[string](debounceDelay)
			var last atomic.Int32
			var calls atomic.Int32
			for i := range debounceBurst {
				k.Do("doc", func() {
					last.Store(int32(i))
					calls.Add(1)
				})
				time.Sleep(debounceDelay / 2)
			}
			time.Sleep(debounceDelay)
			synctest.Wait()
			assert.EqualValues(t, 1, calls.Load())
			assert.EqualValues(t, debounceBurst-1, last.Load())
		})
	})
	t.Run("keys debounce independently", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			k := debounce.NewKeyed[int](debounceDelay)
			var calls atomic.Int32
			k.Do(1, func() {
				calls.Add(1)
			})
			k.Do(2, func() {
				calls.Add(1)
			})
			time.Sleep(debounceDelay)
			synctest.Wait()
			assert.EqualValues(t, 2, calls.Load())
		})
	})
	t.Run("cancel drops a waiting call", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			k := debounce.NewKeyed[int](debounceDelay)
			var calls atomic.Int32
			k.Do(1, func() {
				calls.Add(1)
			})
			k.Cancel(1)
			k.Cancel(2)
			time.Sleep(debounceDelay)
			synctest.Wait()
			assert.Zero(t, calls.Load())
		})
	})
	t.Run("stop drops and refuses calls", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			k := debounce.NewKeyed[int](debounceDelay)
			var calls atomic.Int32
			k.Do(1, func() {
				calls.Add(1)
			})
			k.Stop()
			k.Do(2, func() {
				calls.Add(1)
			})
			time.Sleep(debounceDelay)
			synctest.Wait()
			assert.Zero(t, calls.Load())
		})
	})
}
