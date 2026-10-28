package idgen

import "testing"

func TestFuncGenerate(t *testing.T) {
	t.Parallel()

	generator := Func(func() string {
		return "generated-id"
	})

	result := generator.Generate()

	if result != "generated-id" {
		t.Fatalf("Generate() = %q, want %q", result, "generated-id")
	}
}

func TestNilFuncPanics(t *testing.T) {
	t.Parallel()

	var generator Func

	assertPanics(t, func() {
		generator.Generate()
	})
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()

	fn()
}
