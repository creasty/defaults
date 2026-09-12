// Package fixture provides types declared outside the test package, so tests can observe how Set
// behaves across a package boundary.
package fixture

// Sample has one exported and one unexported field. Set can only touch the former, and a test in
// another package can only observe the latter through UnexportedField.
type Sample struct {
	ExportedField int `default:"1"`

	// Tagged on purpose: without a tag, an assertion that this stays zero cannot fail, because
	// nothing would have set it even if reflect were allowed to.
	unexportedField int `default:"1"`
}

// UnexportedField returns the field Set is expected to leave alone.
func (s Sample) UnexportedField() int {
	return s.unexportedField
}
