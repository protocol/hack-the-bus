package eventbus

import "context"

// Terminology

// A bus is single strongly ordered queue of events.

// Publishers publish events to a bus. The order of published events is always
// strongly consistent. If two separate publishers publish an event
// at the same time, one call will block until the other is able to finish
// putting the event n the queue, so that the ordering of events is strongly consistent.

// Consumers consume events on a bus. Consumers describe to the bus which events
// they will consume, and this library handles skipping over events that are not
// of interest to the consumer. Consumption of events is generally asynchronous,
// so that position of the next event to be consumed in the bus's queue is only
// eventually consistent among the subscribers.
//
// However, a event Publisher can force a synchronization upon publishing an event
// so that the position of next consumable event in the queue becomes strongly
// consistent among all Consumers -- namely at the position immediately following the
// published event. When this happens, Consumers that reach the checkpointed event in
// the queue will receive no more events until all Consumers reach the checkpointed event
//
// Most participants in a bus will be ProducerConsumers, meaning they will both
// publish and consume events
//
// Each Bus has a unique UUID.
//
// Since synchronous events represent a deadlock risk, each bus is setup with a
// failure policy -- this defines the conditions under which a particular consumer
// is considered to be no longer consuming events properly. This can be maximum time
// waiting for a synchronous event to be processed, or a maximum number events behind
// the current position. When this occurs a recovery handler is called that can
// decide what to do -- they may evict the consumer as a participant, or fail
// the whole bus.
//
// Buses also have observers. Observers do not participate in the event consumption process
// but are periodically notified of latest events

// Participant is a producer or consumer in a bus
type BusParticipant interface {
	// BusUUID returns the uuid of the bus this is a participant for
	BusUUID() UUID
	// ParticipantID returns the ID of this participant
	Participant() Participant
}

// CanConsume is any entity that can consume events on the bus
type CanConsume interface {
	// AddEventInterests adds events types that this consumer is interested in.
	// Events published prior to the current queue position are NOT
	// affected
	AddEventInterests([]EventType)
	// CurrentPosition is order in the queue of the next event
	CurrentPosition() Position
	// NextConsumableEvent returns the next event to be consumed
	// it returns a channel that will emit a single value when the
	// next event is available. If there is an event available already
	// the channel will emit immediately.
	NextConsumableEvent() <-chan ConsumableEventResult
	// RemoveEventInterests adds event types that this consumer is interested in.
	// Events published prior to the current queue position are NOT
	// affected
	RemoveEventInterests([]EventType) error
}

// CanPublish is an entity that can publish events on a bus
type CanPublish interface {
	// PublishEvent publishes an event on the bus to be consumed asynchronously
	// by consumers.
	// It will block until any calls to PublishEvent that are in progress finish and
	// then return after the event is published in sequence after those calls
	// If a PublishSynchronousEvent is in progress, PublishEvent will block until PublishSynchronousEvent
	// puts the event in the queue, but not until the consumers catch up. It will then publish the event
	// in the queue. But the event will not become available to consumers until all of them catch up to
	// the synchronization checkpoint.
	// In other words, you can still publish while a synchronization is occurring, but no one sees your
	// event until everyone is synchronized
	PublishEvent(EventType, EventData) error
	// PublishSynchronousEvent publishes an event an sets it up as a synchronization checkpoint
	// It will not return until the event is published AND all consumers catch up to the synchronization
	// checkpoint.
	PublishSynchronousEvent(EventType, EventData) error
}

// Producer is a participant in a bus that publishes events
type Producer interface {
	BusParticipant
	CanPublish
}

// Consumer is a participant in a bus that consumes events
type Consumer interface {
	BusParticipant
	CanConsume
}

// Bus is an individual bus that maintains a single order of events
type Bus interface {
	UUID()
	// CreateProducerConsumer creates a new producer consumer for a bus
	// It will error if a producer participant of the same type exists for this bus
	// If this bus was created from a collection, it will error if the particpant is
	// not unique to the collection
	CreateProducer(Participant) (Producer, error)
	// CreateConsumer creates a consumer for a bus
	// It will error if a consumer participant of the same type exists for this bus
	// If this bus was created from a collection, it will error if the particpant is
	// not unique to the collection
	CreateConsumer(Participant) (Consumer, error)
	// LookupProducer returns am existing Producer
	LookupProducer(Participant) (Producer, error)
	// LookupConsumer returns an existing Consumer
	LookupConsumer(Participant) (Consumer, error)
	// AddObserver adds a observer for the given event types for this bus
	// It replays whatever events are still in an in memory or on disk queue for the given bus
	AddObserver(eventTypes []EventType, observer Observer)
	// SetFailureHandler sets the failure handler for the bus
	// This is called when the bus see a failure to process events for a given
	// participant as defined by the buses failure policy
	SetFailureHandler(FailureHandler)
}

// Collection is a group of event buses that each maintain their own independent strongly
// consistent queue of events.
//
// We use the abstract term Session to define the the scope under which all events share a single bus.
// Delineating where to draw the boundaries for a Session is up to the consumer of this library
//
// (as a concrete example, a "Session" in the context of Filecoin markets
// is a single complete data transfer)
//
// TBD: do we have multiple Collections? I don't think we do. The intent is to allow
// configurability such that different buses behave differently but still might share
// common participants (so that data transfer for example would not have to know about
// multiple collections for storage and retrieval)
type Collection interface {
	// CreateNewBus creates a new Bus and returns its UUID
	// It takes a list of policies under which it will declare a consumer "failed"
	CreateNewBus(failurePolicy []FailureCondition) (Bus, error)
	// GetByUUID returns the bus with the given UUID
	GetByUUID(UUID) (Bus, error)
	// GetBusByParticipant find the bus that contains the unique particpant
	// (useful for relocating a bus without having to find a UUID)
	GetBusByParticpant(Participant) (Bus, error)

	// AddObserver adds an observer for the given event types across all buses.
	// Observers have different characteristics to consumers
	// They simply receive periodic updates about events that have happened
	// The only guarantees are:
	// - they receive events in order groups per bus (no guarantee about ordering between buses)
	// - they ALWAYS receive events after ALL consumers consume or skip them
	// For AddObserver, it only observes events published after it's added
	AddObserver(eventTypes []EventType, observer Observer)

	// DeleteBus shuts down a bus. Since buses are relatively transient (length of a transfer), we should be able
	// to shut down and throw away a bus and all its events. Any existing ProducerConsumer/Producer/Consumer instances
	// will just return errors on their methods
	DeleteBus(context.Context, UUID) error

	// SetDefaultFailureHandler sets the default failure handler for all newly created buses
	SetDefaultFailureHandler(FailureHandler)
}

// Persistence thoughts:
// Honestly I wonder if we should just persist everything on disk and then delete the log
// when the bus is no longer needed. There's a lot of reasons this might make sense
// It would allow for predictable resumability and replayability
// But since we only care about these logs while the bus is "running", we could delete them

// UUID is simply a uuid as a string, generated by the uuid library
type UUID string

// Position is a logical position for an event in a bus's even queue
// The first event published is 0, the next is 1, then 2,3,4 and so on
// You can think of Position as describing relative time ordering within a
// given bus
type Position uint64

// EventType describes a kind of event
// event types are generally used to delineate what events a consumer/observer is interested in
type EventType interface{}

// EventData describes the actual content of an event
// It can be any arbitrary data
type EventData interface{}

// Event is an event in a bus queue
type Event interface {
	BusUUID() UUID
	Position() Position
	Publisher() Participant
	Type() EventType
	Data() EventData
}

// ConsumableEvent is an event that can be marked consumed
type ConsumableEvent interface {
	Event
	MarkConsumed()
}

// ConsumableEventResult is the result type of calling NextConsumableEvent
type ConsumableEventResult interface {
	Event() ConsumableEvent
	Error() error
}

// FailureCondition is a condition on which
// TBD: need to think about what this actually looks like
type FailureCondition interface{}

// Observer is just a callback that is called with a set of recently published events
// It may not get called on every event and instead may send multiple events at once
type Observer func([]Event)

// Participant is a unique producer/consumer for a specific bus instance
//
// The type identifies the name of the external service producing/consuming events
// A single service should use the same name for all the buses it produces/consumes
// events on
//
// The LocalID identifies the local state managed by the service that is associated
// with this specific bus (example: if the service maintains a collection of state, and each
// instance of state in the collection is associated with one event bus, the local id
// might be an index into the collection)
// A single service would use a different and hopefully unique LocalID for every
// bus it interacts with
//
// A Participant (containing both type and local id) should be able to uniquely
// identified a bus in a collection
type Participant struct {
	Type    ParticipantType
	LocalID ParticipantLocalID
}

// ParticipantType is an identifier for a given participant in a queue
type ParticipantType interface{}

// ParticipantLocalID is an piece of data a given participant can use to associate the
// given bus with it's own local data or state
type ParticipantLocalID interface{}

// FailureHandler receives failures for a bus and decides what to do with it
type FailureHandler interface {
	HandleFailure(UUID, Participant, error, RecoveryOptions)
}

// RecoveryOptions are actions a failure handler can choose to take
// when a consumer fails
type RecoveryOptions interface {
	EvictParticipant() error
	FailBus() error
}
