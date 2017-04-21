package defaults

// DefaultsSetter is an interface for setting default values
type DefaultsSetter interface {
	SetDefaults()
}

func callDefaultsSetter(v interface{}) {
	if ds, ok := v.(DefaultsSetter); ok {
		ds.SetDefaults()
	}
}
