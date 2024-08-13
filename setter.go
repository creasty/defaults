package defaults

// Setter is an interface for setting default values
type Setter interface {
	SetDefaults()
}

func callSetter(v interface{}) {
	if ds, ok := v.(Setter); ok {
		ds.SetDefaults()
	}
}

type TaggedSetter interface {
	SetTaggedDefaults(tag string) error
}

func callTaggedSetter(v interface{}, tag string) error {
	if ds, ok := v.(TaggedSetter); ok {
		return ds.SetTaggedDefaults(tag)
	}
	return nil
}
