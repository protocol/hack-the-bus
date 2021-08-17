package eventbus

import (
	"fmt"
	"sync"
)

var _ Consumer = (*defaultConsumer)(nil)

type defaultConsumer struct {
	bus       *defaultBus
	par       Participant
	interests map[EventType]struct{}
	q         chan NextEvents
	mu        sync.RWMutex
	commPos   Position
	currPos   Position
}

func (c *defaultConsumer) Participant() Participant {
	return c.par
}

func (c *defaultConsumer) Bus() Bus {
	return c.bus
}

func (c *defaultConsumer) AddEventInterests(et []EventType) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range et {
		c.interests[e] = struct{}{}
	}
	return nil
}

func (c *defaultConsumer) RemoveEventInterests(et []EventType) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range et {
		delete(c.interests, e)
	}
	return nil
}

func (c *defaultConsumer) CommittedPosition() Position {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.commPos
}

func (c *defaultConsumer) Position() Position {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currPos
}

func (c *defaultConsumer) NextEvents() <-chan NextEvents {
	return c.q
}

func (c *defaultConsumer) Commit(pos Position) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.commPos > pos {
		return fmt.Errorf("cannot commit position to %v since it is before curent commit position %v", pos, c.commPos)
	}
	c.commPos = pos
	return nil
}

func (c *defaultConsumer) dispatch(evts NextEvents) {
	if evts.Error() != nil {
		c.q <- evts
		return
	}
	ne := &defaultNextEvents{}
	for _, e := range evts.Events() {
		if _, ok := c.interests[e.Type()]; ok {
			ne.evts = append(ne.evts, e)
		}
	}
	if len(ne.evts) != 0 {
		c.q <- ne
		// TODO update current position
	}
	return
}
