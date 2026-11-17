package notice

// Actor is who did something: an Account's display name, or the API key.
type Actor struct {
	Name   string `json:"name"`
	APIKey bool   `json:"apiKey"`
}

// Subject words the actor as a sentence's subject: "Sam", "An app using
// the API key".
func (a Actor) Subject() string {
	if a.APIKey {
		return "An app using the API key"
	}
	return a.Name
}
