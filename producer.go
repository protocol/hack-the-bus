package eventbus

var _ Producer = (*defaultProducer)(nil)

type defaultProducer struct {
	bus *defaultBus
	par Participant
}

func (p *defaultProducer) Participant() Participant {
	return p.par
}

func (p *defaultProducer) Bus() Bus {
	return p.bus
}

func (p *defaultProducer) PublishEvent(et EventType, data EventData) <-chan error {
	published := make(chan error)
	go func() {
		evt := &defaultEvent{
			uuid: p.Bus().UUID(),
			par:  p.par,
			et:   et,
			data: data,
		}
		published <- p.bus.publish(evt)
	}()
	return published
}

func (p *defaultProducer) PublishSynchronousEvent(et EventType, data EventData) (<-chan error, <-chan error) {
	panic("implement me")
}
