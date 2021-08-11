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
type Participant interface {
	// BusUUID returns the uuid of the bus this is a participant for
	BusUUID() UUID
	// ParticipantID returns the ID of this participant
	ParticipantID() ParticipantID
	// ParticipantKey returns the key for this bus registered by the participant.
	// The key is an identifier for the bus that has meaning for the participant
	// -- used to associate the bus to other code the participant is working with
	ParticipantKey() ParticipantKey
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
	Participant
	CanPublish
}

// Consumer is a participant in a bus that consumes events
type Consumer interface {
	Participant
	CanConsume
}

// ProducerConsumer is a participant in a bus that publishes and consumes events
type ProducerConsumer interface {
	Participant
	CanPublish
	CanConsume
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
	CreateNewBus(failurePolicy []FailureCondition) (UUID, error)
	// AddProducerConsumer adds a new ProducerConsumer to a bus with the given ParticipantID and ParticipantKey
	// If the given bus already has the given ParticipantID registered, this method will error
	// It returns the interface to produce and consume events
	AddBusProducerConsumer(busUUID UUID, participantID ParticipantID, participant ParticipantKey) (ProducerConsumer, error)
	// TBD: Do we need these methods or is everyone a ProducerConsumer???
	// AddProducer adds a new Producer to a bus with the given ParticipantID and ParticipantKey
	// If the given bus already has the given ParticipantID registered, this method will error
	// It returns the interface to produce events
	AddBusProducer(busUUID UUID, participantID ParticipantID, participant ParticipantKey) (Producer, error)
	// AddConsumer adds a new Consumer to a bus with the given ParticipantID and ParticipantKey
	// If the given bus already has the given ParticipantID registered, this method will error
	// It returns the interface to consume events
	AddBusConsumer(busUUID UUID, participantID ParticipantID, participant ParticipantKey) (Consumer, error)
	// GetBusProducerConsumer finds a bus that contains the given ParticipantID and ParticipantKey
	// And returns an interface to produce and consume events
	// This might be used to restart processing of events on restart
	GetBusProducerConsumer(participantID ParticipantID, participant ParticipantKey) (ProducerConsumer, bool, error)
	// GetBusProducer finds a bus that contains the given ParticipantID and ParticipantKey
	// And returns an interface to produce events
	// This might be used to restart processing of events on restart
	GetBusProducer(participantID ParticipantID, participant ParticipantKey) (Producer, bool, error)
	// GetBusConsumer finds a bus that contains the given ParticipantID and ParticipantKey
	// And returns an interface to consume events
	// This might be used to restart processing of events on restart
	GetBusConsumer(participantID ParticipantID, participant ParticipantKey) (Consumer, bool, error)

	// AddObserver adds an observer for the given event types across all buses.
	// Observers have different characteristics to consumers
	// They simply receive periodic updates about events that have happened
	// The only guarantees are:
	// - they receive events in order groups per bus (no guarantee about ordering between buses)
	// - they ALWAYS receive events after ALL consumers consume or skip them
	// For AddObserver, it only observes events published after it's added
	AddObserver(eventTypes []EventType, observer Observer)
	// AddBusObserver adds a observer for the given event types for the given bus
	// It replays whatever events are still in an in memory or on disk queue for the given bus
	AddBusObserver(busUUID UUID, eventTypes []EventType, observer Observer)
	// DeleteBus shuts down a bus. Since buses are relatively transient (length of a transfer), we should be able
	// to shut down and throw away a bus and all its events. Any existing ProducerConsumer/Producer/Consumer instances
	// will just return errors on their methods
	DeleteBus(context.Context, UUID) error

	// SetFailureHandler establishes a failure handler for the given bus
	// This is called when the bus see a failure to process events for a given
	// participant as defined by the buses failure policy
	SetFailureHandler(busUUID UUID, handler FailureHandler)
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
	PublisherID() ParticipantID
	PublisherKey() ParticipantKey
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

// ParticipantID is an identifier for a given participant in a queue
type ParticipantID interface{}

// ParticipantKey is an piece of data a given participant can use to associate the
// given bus with it's own local data or state
type ParticipantKey interface{}

// FailureHandler receives failures for a bus and decides what to do with it
type FailureHandler interface {
	HandleFailure(UUID, ParticipantID, ParticipantKey, error, RecoveryOptions)
}

// RecoveryOptions are actions a failure handler can choose to take
// when a consumer fails
type RecoveryOptions interface {
	EvictParticipant() error
	FailBus() error
}
