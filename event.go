package zkwatcher

// EventType enum type.
type EventType int

// EventType enum values.
const (
	_ EventType = iota
	Create
	Delete
	Update
)

func (e EventType) String() string {
	switch e {
	case Create:
		return "create"
	case Delete:
		return "delete"
	case Update:
		return "update"
	default:
		return "unkown"
	}
}

// Event represent a change in the watched tree.
type Event struct {
	Path  string
	Type  EventType
	Error error
}
