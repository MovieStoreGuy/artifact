package artifact

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/tilinna/clock"
)

var (
	// ErrNilValue is used when a nil paramater is provided
	ErrNilValue = errors.New("nil value provided")
)

// RWLocker is an extension of the sync.Locker
// to include the read lock methods.
type RWLocker interface {
	sync.Locker

	RLock()
	RUnlock()
}

// Artifact allows other components to be notified when
// this type has changed
type Artifact interface {
	// RWLocker is required to ensure go routine safety.
	RWLocker

	// ModifiedAt returns the time that the artifact
	// was last updated.
	ModifiedAt() time.Time

	// NotifyUpdate will inform all registered listeners
	// that a change has been made and will also update
	// modified time for the type.
	NotifyUpdated(ctx context.Context)

	// Register stores the information about the channel
	// and will send updates once Update is called successfully
	Register(notify chan<- struct{}) (err error)

	// Update must read the incoming reader to then
	// apply the update to the type, any issue will
	// return ar error and not notifications will happen.
	//
	// Note: Callee's a requried to call `Artifact.Lock` and `Artifact.Unlock`
	// respectively, they must not be called within the update method.
	Update(in io.Reader) error
}

// Notifier is a partial implementation of Artifact
// so that it can be embedded into types to help
// reduce code duplication
type Notifier struct {
	sync.RWMutex
	mod      time.Time
	notifies []chan<- struct{}
}

func (n *Notifier) ModifiedAt() time.Time {
	n.RLock()
	defer n.RUnlock()

	return n.mod
}

func (n *Notifier) Register(notify chan<- struct{}) error {
	if notify == nil {
		return fmt.Errorf("notify: %w", ErrNilValue)
	}

	n.Lock()
	defer n.Unlock()

	n.notifies = append(n.notifies, notify)
	return nil
}

func (n *Notifier) NotifyUpdated(ctx context.Context) {
	n.Lock()
	n.mod = clock.FromContext(ctx).Now()
	n.Unlock()

	n.RLock()
	defer n.RUnlock()

	for _, notify := range n.notifies {
		select {
		case notify <- struct{}{}:
		default:
			// In the event that the updates are happening faster than
			// what the requested notify can read the type
			// so we just cary on.
		}
	}
}
