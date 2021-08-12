package eventbus

import (
	"context"
)

type defaultBus struct {
	id  UUID
	fc  []FailureCondition
	fh  FailureHandler
	obs []*observerWithInterests // TODO we probably want to deduplicate entries here.
}

func (b *defaultBus) UUID() UUID {
	return b.id
}

func (b *defaultBus) CreateProducer(p Participant) (Producer, error) {
	panic("implement me")
}

func (b *defaultBus) CreateConsumer(p Participant) (Consumer, error) {
	panic("implement me")
}

func (b *defaultBus) LookupProducer(p Participant) (Producer, error) {
	panic("implement me")
}

func (b *defaultBus) LookupConsumer(p Participant) (Consumer, error) {
	panic("implement me")
}

func (b *defaultBus) AddObserver(et []EventType, o Observer) {
	b.obs = append(b.obs, &observerWithInterests{et: et, o: o})
}

func (b *defaultBus) SetFailureHandler(fh FailureHandler) {
	b.fh = fh
}

// TODO do we need to add this to the Bus interface?
func (b *defaultBus) hasParticipant(p Participant) bool {
	panic("implement me")
}

// TODO do we need to add this to the Bus interface?
func (b *defaultBus) delete(context.Context) error {
	panic("implement me")
}
