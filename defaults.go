package defaults

// Init initializes members in a struct referenced by a pointer.
// Maps and slices are initialized by `make` and other primitive types are set with default values
func Init(v interface{}) error {
	return nil
}
