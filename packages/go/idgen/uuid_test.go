package idgen

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestUUIDGenerate(t *testing.T) {
	t.Parallel()

	generator := NewUUID()

	result := generator.Generate()

	parsed, err := uuid.Parse(result)
	if err != nil {
		t.Fatalf("Parse(%q): %v", result, err)
	}

	if parsed.Version() != 7 {
		t.Fatalf("Version() = %d, want 7", parsed.Version())
	}

	if parsed.Variant() != uuid.RFC4122 {
		t.Fatalf("Variant() = %v, want %v", parsed.Variant(), uuid.RFC4122)
	}
}

func TestUUIDNew(t *testing.T) {
	t.Parallel()

	result := NewUUID().New()

	if result.Version() != 7 {
		t.Fatalf("Version() = %d, want 7", result.Version())
	}

	if result.Variant() != uuid.RFC4122 {
		t.Fatalf("Variant() = %v, want %v", result.Variant(), uuid.RFC4122)
	}
}

func TestUUIDGenerateUniqueValues(t *testing.T) {
	t.Parallel()

	generator := NewUUID()

	first := generator.Generate()
	second := generator.Generate()

	if first == second {
		t.Fatalf("Generate() returned duplicate value %q", first)
	}
}

func TestUUIDConcurrentUse(t *testing.T) {
	t.Parallel()

	const count = 1000

	generator := NewUUID()
	values := make(chan string, count)

	var wg sync.WaitGroup
	wg.Add(count)

	for range count {
		go func() {
			defer wg.Done()

			values <- generator.Generate()
		}()
	}

	wg.Wait()
	close(values)

	unique := make(map[string]struct{}, count)

	for value := range values {
		if _, exists := unique[value]; exists {
			t.Fatalf("duplicate value %q", value)
		}

		unique[value] = struct{}{}
	}

	if len(unique) != count {
		t.Fatalf("generated %d unique values, want %d", len(unique), count)
	}
}

func TestUUIDImplementsGenerator(t *testing.T) {
	t.Parallel()

	var _ Generator = NewUUID()
}
