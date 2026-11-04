package idgen

import (
	"strconv"
	"sync"
	"testing"
)

func TestSequenceGenerate(t *testing.T) {
	t.Parallel()

	generator := NewSequence(
		"first-id",
		"second-id",
		"third-id",
	)

	tests := []struct {
		name string
		want string
	}{
		{
			name: "first",
			want: "first-id",
		},
		{
			name: "second",
			want: "second-id",
		},
		{
			name: "third",
			want: "third-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.Generate()

			if result != tt.want {
				t.Fatalf("Generate() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestSequenceRemaining(t *testing.T) {
	t.Parallel()

	generator := NewSequence(
		"first-id",
		"second-id",
	)

	if result := generator.Remaining(); result != 2 {
		t.Fatalf("Remaining() = %d, want 2", result)
	}

	generator.Generate()

	if result := generator.Remaining(); result != 1 {
		t.Fatalf("Remaining() = %d, want 1", result)
	}

	generator.Generate()

	if result := generator.Remaining(); result != 0 {
		t.Fatalf("Remaining() = %d, want 0", result)
	}
}

func TestSequenceAdd(t *testing.T) {
	t.Parallel()

	generator := NewSequence("first-id")

	if result := generator.Generate(); result != "first-id" {
		t.Fatalf("Generate() = %q, want %q", result, "first-id")
	}

	generator.Add("second-id", "third-id")

	if result := generator.Generate(); result != "second-id" {
		t.Fatalf("Generate() = %q, want %q", result, "second-id")
	}

	if result := generator.Generate(); result != "third-id" {
		t.Fatalf("Generate() = %q, want %q", result, "third-id")
	}
}

func TestSequenceExhaustedPanics(t *testing.T) {
	t.Parallel()

	generator := NewSequence("entity-id")
	generator.Generate()

	assertPanics(t, func() {
		generator.Generate()
	})
}

func TestNewSequenceWithEmptyValuePanics(t *testing.T) {
	t.Parallel()

	assertPanics(t, func() {
		NewSequence("first-id", "", "third-id")
	})
}

func TestSequenceAddEmptyValuePanics(t *testing.T) {
	t.Parallel()

	generator := NewSequence()

	assertPanics(t, func() {
		generator.Add("")
	})
}

func TestNilSequenceGeneratePanics(t *testing.T) {
	t.Parallel()

	var generator *Sequence

	assertPanics(t, func() {
		generator.Generate()
	})
}

func TestNilSequenceAddPanics(t *testing.T) {
	t.Parallel()

	var generator *Sequence

	assertPanics(t, func() {
		generator.Add("entity-id")
	})
}

func TestNilSequenceRemainingPanics(t *testing.T) {
	t.Parallel()

	var generator *Sequence

	assertPanics(t, func() {
		generator.Remaining()
	})
}

func TestSequenceConcurrentUse(t *testing.T) {

	t.Parallel()
	const count = 100
	values := make([]string, count)
	for i := range values {
		values[i] = "id-" + strconv.Itoa(i)
	}
	generator := NewSequence(values...)
	results := make(chan string, count)
	var wg sync.WaitGroup
	wg.Add(count)
	for range count {
		go func() {
			defer wg.Done()
			results <- generator.Generate()
		}()
	}
	wg.Wait()
	close(results)
	unique := make(map[string]struct{}, count)
	for value := range results {
		if _, exists := unique[value]; exists {
			t.Fatalf("duplicate value %q", value)
		}
		unique[value] = struct{}{}
	}
	if len(unique) != count {
		t.Fatalf("generated %d unique values, want %d", len(unique), count)
	}
	if remaining := generator.Remaining(); remaining != 0 {
		t.Fatalf("Remaining() = %d, want 0", remaining)
	}

}
func TestSequenceImplementsGenerator(t *testing.T) {
	t.Parallel()

	var _ Generator = NewSequence("entity-id")
}
