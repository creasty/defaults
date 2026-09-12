package defaults

// MustSet function is a wrapper of Set function
// It will call Set and panic if err not equals nil.
func MustSet(ptr interface{}) {
	if err := Set(ptr); err != nil {
		panic(err)
	}
}
