package defaults_test

import (
	"testing"
	"time"

	"github.com/creasty/defaults"
)

// BenchmarkSet measures the whole walk: every field is visited, checked for whether it still holds
// its zero value, and parsed into. The zero check runs once per field, so it is the part of the cost
// that scales with the struct rather than with the tags.
func BenchmarkSet(b *testing.B) {
	b.Run("scalars", func(b *testing.B) {
		type st struct {
			Str  string        `default:"str"`
			Int  int           `default:"1"`
			Bool bool          `default:"true"`
			Dur  time.Duration `default:"5s"`
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var got st
			if err := defaults.Set(&got); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("composites", func(b *testing.B) {
		type inner struct {
			Name string `default:"inner"`
			Port int    `default:"8080"`
		}
		type st struct {
			Inner inner
			Ptr   *inner            `default:"{}"`
			Slice []string          `default:"[\"a\",\"b\"]"`
			Map   map[string]string `default:"{\"k\":\"v\"}"`
		}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var got st
			if err := defaults.Set(&got); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkCanUpdate measures the zero check on its own, which is what Set spends per field.
func BenchmarkCanUpdate(b *testing.B) {
	b.Run("scalar", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !defaults.CanUpdate(0) {
				b.Fatal("a zero int is updatable")
			}
		}
	})

	b.Run("struct", func(b *testing.B) {
		type st struct {
			Str   string
			Int   int
			Slice []string
			Map   map[string]int
		}

		var zero st

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if !defaults.CanUpdate(zero) {
				b.Fatal("a zero struct is updatable")
			}
		}
	})
}
