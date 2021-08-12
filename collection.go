package eventbus

import (
	"context"
	"fmt"
	"github.com/google/uuid"
)

type defaultCollection struct {
	buses map[UUID]*defaultBus
	dfh   FailureHandler
	obs   []*observerWithInterests // TODO we probably want to deduplicate entries here.
}

func NewCollection() Collection {
	return &defaultCollection{}
}

func (c *defaultCollection) CreateNewBus(fc []FailureCondition) (Bus, error) {
	uid, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	id := UUID(uid.String())
	bus := &defaultBus{
		id: id,
		fc: fc,
		fh: c.dfh,
	}
	// Add existing observers, if any.
	for _, o := range c.obs {
		bus.AddObserver(o.et, o.o)
	}
	c.buses[id] = bus
	return bus, nil
}

func (c *defaultCollection) GetByUUID(uuid UUID) (Bus, error) {
	return c.getByUUID(uuid)
}

func (c *defaultCollection) getByUUID(uuid UUID) (*defaultBus, error) {
	bus, ok := c.buses[uuid]
	if !ok {
		return nil, fmt.Errorf("bus not found with UUID: %v", uuid)
	}
	return bus, nil
}

func (c *defaultCollection) GetBusByParticipant(p Participant) (Bus, error) {
	for _, bus := range c.buses {
		if bus.hasParticipant(p) {
			return bus, nil
		}
	}
	return nil, fmt.Errorf("no bus found for participant: %v", p)
}

func (c *defaultCollection) AddObserver(et []EventType, o Observer) {
	c.obs = append(c.obs, &observerWithInterests{et: et, o: o})
	for _, bus := range c.buses {
		bus.AddObserver(et, o)
	}
}

func (c *defaultCollection) DeleteBus(ctx context.Context, uuid UUID) error {
	bus, err := c.getByUUID(uuid)
	if err != nil {
		return err
	}
	if err := bus.delete(ctx); err != nil {
		return err
	}
	delete(c.buses, bus.UUID())
	return nil
}

func (c *defaultCollection) SetDefaultFailureHandler(fh FailureHandler) {
	c.dfh = fh
}
