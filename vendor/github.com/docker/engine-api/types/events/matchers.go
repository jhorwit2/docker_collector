package events

// Matcher helps match events based on certain criteria for filtering events.
type Matcher interface {
	Matches(event Message) bool
}

// MatcherFunc return true if the event matches or false if not.
type MatcherFunc func(event Message) bool

// Matches returns true if event matches; otherwise, returns false.
func (fn MatcherFunc) Matches(event Message) bool {
	return fn(event)
}

// TypeMatcher returns a matcher to check if the event type
// equals the given type.
func TypeMatcher(eventType string) MatcherFunc {
	return func(event Message) bool {
		return event.Type == eventType
	}
}

// ActionMatcher returns a matcher to check if the event action
// equals the given action.
func ActionMatcher(action string) MatcherFunc {
	return func(event Message) bool {
		return event.Action == action
	}
}

// ActorMatcher returns a matcher to check if the event type
// equals the given type.
func ActorMatcher(id string) MatcherFunc {
	return func(event Message) bool {
		return event.Actor.ID == id
	}
}

// HasAttributeMatcher returns a matcher to check if the
// event has the given attribute regardless of the value.
func HasAttributeMatcher(attribute string) MatcherFunc {
	return func(event Message) bool {
		attributes := event.Actor.Attributes
		if attributes == nil {
			return false
		}

		_, ok := attributes[attribute]
		return ok
	}
}

// AttributeMatcher returns a matcher to check if the event
// has the given attribute with the given value.
func AttributeMatcher(attribute, value string) MatcherFunc {
	return func(event Message) bool {
		attributes := event.Actor.Attributes
		if attributes == nil {
			return false
		}

		v, ok := attributes[attribute]
		return ok && v == value
	}
}
