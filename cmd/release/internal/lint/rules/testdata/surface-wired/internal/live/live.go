package live

type Event interface{ Name() string }

func AdminEvent[T any](name string) Event    { return nil }
func AccountEvent[T any](name string) Event  { return nil }
func EveryoneEvent[T any](name string) Event { return nil }

type Resync struct{}

var ResyncEvent = EveryoneEvent[Resync]("live.resync")
