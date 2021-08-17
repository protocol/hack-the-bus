package eventbus

import (
	"context"
	"fmt"
	"sync"
)

var _ Bus = (*defaultBus)(nil)

type defaultBus struct {
	id        UUID
	fc        []FailureCondition
	fh        FailureHandler
	obs       []*observerWithInterests // TODO we probably want to deduplicate entries here.
	prds      map[Participant]*defaultProducer
	cons      map[Participant]*defaultConsumer
	q         chan Event
	latestPos Position
	mu        sync.RWMutex
}

func newDefaultBus(id UUID, fc []FailureCondition, maxInFlightEvents int) *defaultBus {
	bus := &defaultBus{
		id:   id,
		fc:   fc,
		q:    make(chan Event, maxInFlightEvents),
		prds: make(map[Participant]*defaultProducer),
		cons: make(map[Participant]*defaultConsumer),
	}

	go func() {
		for nextEvt := range bus.q {

			// TODO implement batching; for now one at a time.
			ne := &defaultNextEvents{
				evts: []Event{nextEvt},
				err:  nil,
			}

			for _, c := range bus.cons {
				c.dispatch(ne)
			}
		}
	}()

	return bus
}

func (b *defaultBus) UUID() UUID {
	return b.id
}

func (b *defaultBus) CreateProducer(p Participant) (Producer, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.hasParticipant(p) {
		return nil, fmt.Errorf("already has participant: %v", p)
	}

	// TODO check that p is not participating in any other bus in the collection

	producer := &defaultProducer{bus: b, par: p}
	b.prds[p] = producer
	return producer, nil
}

func (b *defaultBus) CreateConsumer(p Participant) (Consumer, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.hasParticipant(p) {
		return nil, fmt.Errorf("already has participant: %v", p)
	}

	// TODO check that p is not participating in any other bus in the collection

	consumer := &defaultConsumer{
		bus:       b,
		par:       p,
		interests: make(map[EventType]struct{}),
		q:         make(chan NextEvents),
	}
	b.cons[p] = consumer
	return consumer, nil
}

func (b *defaultBus) LookupProducer(p Participant) (Producer, error) {
	if prd, ok := b.prds[p]; ok {
		return prd, nil
	}
	return nil, fmt.Errorf("no producer found for participant: %v", p)
}

func (b *defaultBus) LookupConsumer(p Participant) (Consumer, error) {
	if con, ok := b.cons[p]; ok {
		return con, nil
	}
	return nil, fmt.Errorf("no consumer found for participant: %v", p)
}

func (b *defaultBus) AddObserver(et []EventType, o Observer) error {
	b.obs = append(b.obs, &observerWithInterests{et: et, o: o})
	return nil
}

func (b *defaultBus) SetFailureHandler(fh FailureHandler) {
	b.fh = fh
}

// TODO do we need to add this to the Bus interface?
func (b *defaultBus) hasParticipant(p Participant) bool {
	_, hasPrd := b.prds[p]
	_, hasCon := b.cons[p]
	return hasPrd || hasCon
}

// TODO do we need to add this to the Bus interface?
func (b *defaultBus) delete(context.Context) error {
	panic("implement me")
}

func (b *defaultBus) publish(evt *defaultEvent) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.latestPos++
	evt.pos = b.latestPos
	b.q <- evt
	// TODO return error when bus is shut down and such
	return nil
}
