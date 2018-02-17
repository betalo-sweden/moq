package p

// PersonStore stores people.
type PersonStore interface {
	Get() *Person
}

// Person is a person.
type Person struct {
	ID      string
	Name    string
	Company string
	Website string
}
