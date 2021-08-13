package eventbus

type observerWithInterests struct {
	et []EventType
	o  Observer
}

func (owi *observerWithInterests) interested(eventType EventType) bool {
	//TODO do we need to optimise with a map?
	for _, e := range owi.et {
		if e == eventType {
			return true
		}
	}
	return false
}
