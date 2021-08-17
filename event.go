package eventbus

var (
	_ Event      = (*defaultEvent)(nil)
	_ NextEvents = (*defaultNextEvents)(nil)
)

type (
	defaultEvent struct {
		uuid UUID
		pos  Position
		par  Participant
		et   EventType
		data EventData
	}
	defaultNextEvents struct {
		evts []Event
		err  error
	}
)

func (e *defaultEvent) BusUUID() UUID {
	return e.uuid
}

func (e *defaultEvent) Position() Position {
	return e.pos
}

func (e *defaultEvent) Publisher() Participant {
	return e.par
}

func (e *defaultEvent) Type() EventType {
	return e.et
}

func (e *defaultEvent) Data() EventData {
	return e.data
}

func (ne *defaultNextEvents) Events() []Event {
	return ne.evts
}

func (ne *defaultNextEvents) Error() error {
	return ne.err
}
