package eventbus_test

import (
	"fmt"
	eventbus "github.com/protocol/hack-the-bus"
	"sync"
)

// ExampleNewCollection demonstrates dissemination events via a bus with multiple participants with
// distinct and overlapping interests. It adds a total of four participants to a bus, one producer
// and three consumers. Two if the consumers have distinct interests, while the third consumer is
// interested in all event types. It then publishes only two events with distinct event types and
// shows a total of four events are consumed by the three consumers: one event by each of the
// consumers with distinct interest, and two events by the consumer that is interested in all event
// types.
func ExampleNewCollection() {
	// TODO expand example with observers once implemented

	interest1 := "fish"
	data1 := "in da sea"
	interest2 := "lobster"
	data2 := "unda da sea"

	collection := eventbus.NewCollection()
	//TODO expand example with failure conditions once it is implemented
	bus, err := collection.CreateNewBus([]eventbus.FailureCondition{})
	if err != nil {
		panic(err)
	}
	fmt.Println("created a bus")

	// Create one consumer with one interest
	cons1Par := eventbus.Participant{
		Type:    "fisherman",
		LocalID: "Boaty McBoatface",
	}
	consumer1, err := bus.CreateConsumer(cons1Par)
	if err != nil {
		panic(err)
	}
	err = consumer1.AddEventInterests([]eventbus.EventType{interest1})
	if err != nil {
		panic(err)
	}
	fmt.Printf("created consumer #1 with interest in %v\n", interest1)

	// Create another consumer with another interest
	cons2Par := eventbus.Participant{
		Type:    "lobstermuncher",
		LocalID: "evergreen",
	}
	consumer2, err := bus.CreateConsumer(cons2Par)
	if err != nil {
		panic(err)
	}
	err = consumer2.AddEventInterests([]eventbus.EventType{interest2})
	if err != nil {
		panic(err)
	}
	fmt.Printf("created consumer #2 with interest in %v\n", interest2)

	// Create yet another consumer with overlapping interest
	cons3Par := eventbus.Participant{
		Type:    "catchemall",
		LocalID: "Sea Señor",
	}
	consumer3, err := bus.CreateConsumer(cons3Par)
	if err != nil {
		panic(err)
	}
	err = consumer3.AddEventInterests([]eventbus.EventType{interest1, interest2})
	if err != nil {
		panic(err)
	}
	fmt.Printf("created consumer #3 with interest in %v and %v\n", interest1, interest2)

	// Create a producer and publish two events, one in each interest
	prodPar := eventbus.Participant{
		Type:    "kaspian",
		LocalID: "sea",
	}
	producer, err := bus.CreateProducer(prodPar)
	if err != nil {
		panic(err)
	}
	fmt.Println("created a producer; publishing two events:")

	<-producer.PublishEvent(interest1, data1)
	fmt.Printf("\tpublished event for consumers interested in %v, with data %v\n", interest1, data1)
	<-producer.PublishEvent(interest2, data2)
	fmt.Printf("\tpublished event for consumers interested in %v with data %v\n", interest2, data2)

	fmt.Println("awaiting consumers to receive events:")
	// Expect 4 events to be consumed by the three consumers:
	// - consumer #1 and #2 should only receive 1 event each corresponding to their interest.
	// - consumer #3 should receive both published events because it is interested in both.
	var wg sync.WaitGroup
	wg.Add(4)
	printEventsWithLabel := func(evts eventbus.NextEvents, label string) {
		err = evts.Error()
		if err != nil {
			panic(err)
		}
		for _, evt := range evts.Events() {
			fmt.Printf("\t%v received event with position %v, event type %v and data %v\n",
				label,
				evt.Position(),
				evt.Type(),
				evt.Data())
			wg.Done()
		}
	}
	go func() {
		for {
			select {
			case evts := <-consumer1.NextEvents():
				printEventsWithLabel(evts, "consumer #1")
			case evts := <-consumer2.NextEvents():
				printEventsWithLabel(evts, "consumer #2")
			case evts := <-consumer3.NextEvents():
				printEventsWithLabel(evts, "consumer #3")
			}

		}
	}()
	wg.Wait()

	// Unordered Output:
	// created a bus
	// created consumer #1 with interest in fish
	// created consumer #2 with interest in lobster
	// created consumer #3 with interest in fish and lobster
	// created a producer; publishing two events:
	// 	published event for consumers interested in fish, with data in da sea
	// 	published event for consumers interested in lobster with data unda da sea
	// awaiting consumers to receive events:
	// 	consumer #1 received event with position 1, event type fish and data in da sea
	// 	consumer #2 received event with position 2, event type lobster and data unda da sea
	// 	consumer #3 received event with position 1, event type fish and data in da sea
	// 	consumer #3 received event with position 2, event type lobster and data unda da sea
}
