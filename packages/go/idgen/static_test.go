package idgen

import "testing"

func TestStaticGenerate(t *testing.T) {
	t.Parallel()

	generator := NewStatic("entity-id")

	first := generator.Generate()
	second := generator.Generate()

	if first != "entity-id" {
		t.Fatalf("first Generate() = %q, want %q", first, "entity-id")
	}

	if second != "entity-id" {
		t.Fatalf("second Generate() = %q, want %q", second, "entity-id")
	}
}

func TestNewStaticWithEmptyValuePanics(t *testing.T) {
	t.Parallel()

	assertPanics(t, func() {
		NewStatic("")
	})
}

func TestZeroStaticPanics(t *testing.T) {
	t.Parallel()

	var generator Static

	assertPanics(t, func() {
		generator.Generate()
	})
}

func TestStaticImplementsGenerator(t *testing.T) {
	t.Parallel()

	var _ Generator = NewStatic("entity-id")
}
