package vacancy

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

var start = time.Date(2026, 11, 27, 11, 0, 0, 0, time.UTC)

func publishableDetails() Details {
	return Details{
		Title: "Инженер", Description: strings.Repeat("я", 50), WorkFormat: WorkRemote,
		Skills: []SkillRequirement{{Key: " Go ", DisplayName: " Go "}},
	}
}

func draft(t *testing.T, details Details) *Vacancy {
	t.Helper()
	v, err := NewDraft(uuid.New(), uuid.New(), uuid.New(), details, start)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestLifecycle(t *testing.T) {
	v := draft(t, publishableDetails())
	original := v.Snapshot()
	if original.Status != StatusDraft || original.Version != 1 || !original.PublishedAt.IsZero() {
		t.Fatalf("unexpected draft: %+v", original)
	}

	type action func(*Vacancy, int64, time.Time) error
	actions := []struct {
		name     string
		run      action
		from, to Status
	}{
		{"publish", (*Vacancy).Publish, StatusDraft, StatusPublished},
		{"archive", (*Vacancy).Archive, StatusPublished, StatusArchived},
		{"restore", (*Vacancy).Restore, StatusArchived, StatusPublished},
		{"close", (*Vacancy).Close, StatusPublished, StatusClosed},
	}
	checkRejections := func() {
		t.Helper()
		before := v.Snapshot()
		for _, a := range actions {
			if err := a.run(v, before.Version-1, before.UpdatedAt); !errors.Is(err, ErrVersionConflict) {
				t.Fatalf("%s accepted stale version or returned wrong error: %v", a.name, err)
			}
			if a.from != before.Status {
				if err := a.run(v, before.Version, before.UpdatedAt); !errors.Is(err, ErrInvalidTransition) {
					t.Fatalf("%s from %s: %v", a.name, before.Status, err)
				}
			}
		}
		if !reflect.DeepEqual(v.Snapshot(), before) {
			t.Fatal("rejected transition mutated the vacancy")
		}
	}

	for i, a := range actions {
		checkRejections()
		version := int64(i + 1)
		now := start.Add(time.Duration(version) * time.Hour)
		if err := a.run(v, version, now); err != nil {
			t.Fatal(err)
		}
		s := v.Snapshot()
		if s.Version != version+1 || s.Status != a.to || !s.UpdatedAt.Equal(now) ||
			s.CreatedAt != start || s.ID != original.ID || s.CompanyID != original.CompanyID || s.AuthorID != original.AuthorID {
			t.Fatalf("unexpected state after %s: %+v", a.name, s)
		}
		reloaded, err := FromState(s)
		if err != nil || !reflect.DeepEqual(reloaded.Snapshot(), s) {
			t.Fatalf("state round trip after %s: %v", a.name, err)
		}
		v = reloaded
	}
	checkRejections()
	s := v.Snapshot()
	if !s.FirstPublishedAt.Equal(start.Add(time.Hour)) || !s.PublishedAt.Equal(start.Add(3*time.Hour)) ||
		!s.ClosedAt.Equal(start.Add(4*time.Hour)) {
		t.Fatalf("unexpected publication history: %+v", s)
	}
	if err := v.UpdateDetails(s.Version, s.Details, s.ClosedAt); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("update closed vacancy: %v", err)
	}
}

func TestPublicationPolicyAndAtomicEdit(t *testing.T) {
	complete := publishableDetails()
	for _, tc := range []struct {
		name string
		edit func(*Details)
	}{
		{"short title", func(d *Details) { d.Title = " тест " }},
		{"short description", func(d *Details) { d.Description = " " + strings.Repeat("я", 49) + " " }},
		{"no skills", func(d *Details) { d.Skills = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			incomplete := complete
			tc.edit(&incomplete)
			v := draft(t, incomplete)
			if err := v.Publish(1, start); !errors.Is(err, ErrNotPublishable) {
				t.Fatalf("publish incomplete draft: %v", err)
			}
			if err := v.UpdateDetails(1, complete, start); err != nil {
				t.Fatal(err)
			}
			if err := v.Publish(2, start); err != nil {
				t.Fatal(err)
			}
			published := v.Snapshot()
			if err := v.UpdateDetails(2, complete, start); !errors.Is(err, ErrVersionConflict) {
				t.Fatalf("stale edit: %v", err)
			}
			if err := v.UpdateDetails(3, incomplete, start); !errors.Is(err, ErrNotPublishable) {
				t.Fatalf("edit published vacancy: %v", err)
			}
			if !reflect.DeepEqual(published, v.Snapshot()) {
				t.Fatal("failed edit changed published content")
			}
			edited := complete
			edited.Title = "Аналитик"
			if err := v.UpdateDetails(3, edited, start); err != nil {
				t.Fatal(err)
			}
			if err := v.Archive(4, start); err != nil {
				t.Fatal(err)
			}
			if err := v.UpdateDetails(5, incomplete, start); err != nil {
				t.Fatal(err)
			}
			archived := v.Snapshot()
			if err := v.Restore(6, start); !errors.Is(err, ErrNotPublishable) {
				t.Fatalf("restore incomplete archive: %v", err)
			}
			if !reflect.DeepEqual(archived, v.Snapshot()) {
				t.Fatal("failed restoration changed archive")
			}
		})
	}
}

func TestSkillsAndValuesAreOwnedByVacancy(t *testing.T) {
	d := publishableDetails()
	location := "Москва"
	minimum := Amount(0)
	d.Location = &location
	d.Compensation = &Compensation{Minimum: &minimum, Currency: "RUB", Period: "MONTH"}
	v := draft(t, d)
	d.Skills[0].Key = "different"
	location = "Казань"
	minimum = 100

	s := v.Snapshot()
	if s.Details.Skills[0] != (SkillRequirement{"go", "Go"}) || *s.Details.Location != "Москва" ||
		*s.Details.Compensation.Minimum != 0 || s.Details.Compensation.Maximum != nil {
		t.Fatal("aggregate aliased inputs or lost the distinction between zero and absent")
	}
	s.Details.Skills = append(s.Details.Skills, SkillRequirement{Key: " GO", DisplayName: "Golang"})
	*s.Details.Compensation.Minimum = 200
	if err := v.UpdateDetails(1, s.Details, start); !errors.Is(err, ErrInvalidDetails) {
		t.Fatalf("duplicate normalized skill: %v", err)
	}
	if got := v.Snapshot(); got.Version != 1 || len(got.Details.Skills) != 1 || *got.Details.Compensation.Minimum != 0 {
		t.Fatal("aggregate changed through a snapshot or rejected edit")
	}
}

func TestCompensation(t *testing.T) {
	zero, one, limit := Amount(0), Amount(1), maxAmount
	negative, overflow := Amount(-1), maxAmount+1
	for _, tc := range []struct {
		name  string
		c     Compensation
		valid bool
	}{
		{"undisclosed", Compensation{}, true},
		{"only upper bound", Compensation{Maximum: &limit, Currency: "EUR", Period: "MONTH", TaxBasis: TaxGross}, true},
		{"exact zero", Compensation{Minimum: &zero, Maximum: &zero, Currency: "RUB", Period: "HOUR", TaxBasis: TaxNet}, true},
		{"negative", Compensation{Minimum: &negative, Currency: "USD", Period: "YEAR"}, false},
		{"over precision limit", Compensation{Maximum: &overflow, Currency: "USD", Period: "YEAR"}, false},
		{"inverted range", Compensation{Minimum: &one, Maximum: &zero, Currency: "EUR", Period: "MONTH"}, false},
		{"missing currency", Compensation{Minimum: &zero, Period: "MONTH"}, false},
		{"missing period", Compensation{Minimum: &zero, Currency: "EUR"}, false},
		{"invalid currency", Compensation{Currency: "usd"}, false},
		{"invalid tax basis", Compensation{TaxBasis: "OTHER"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := publishableDetails()
			d.Compensation = &tc.c
			_, err := NewDraft(uuid.New(), uuid.New(), uuid.New(), d, start)
			if tc.valid && err != nil || !tc.valid && !errors.Is(err, ErrInvalidDetails) {
				t.Fatalf("valid=%t, got %v", tc.valid, err)
			}
		})
	}
}

func TestInvalidState(t *testing.T) {
	v := draft(t, publishableDetails())
	for _, mutate := range []func(*State){
		func(s *State) { s.ID = uuid.Nil },
		func(s *State) { s.CompanyID = uuid.Nil },
		func(s *State) { s.AuthorID = uuid.Nil },
		func(s *State) { s.Version = 0 },
		func(s *State) { s.Status = "DELETED" },
		func(s *State) { s.Status = StatusArchived },
		func(s *State) { s.UpdatedAt = start.Add(-time.Hour) },
		func(s *State) { s.PublishedAt = start },
		func(s *State) { s.ClosedAt = start },
	} {
		s := v.Snapshot()
		mutate(&s)
		if _, err := FromState(s); !errors.Is(err, ErrInvalidVacancy) {
			t.Fatalf("invalid state: %+v, error: %v", s, err)
		}
	}
	for _, mutate := range []func(*Details){
		func(d *Details) { d.Title = " " },
		func(d *Details) { d.WorkFormat = "OTHER" },
		func(d *Details) { d.Skills[0].Key = " " },
		func(d *Details) { d.Skills[0].DisplayName = "" },
	} {
		d := publishableDetails()
		mutate(&d)
		if err := v.UpdateDetails(1, d, start); !errors.Is(err, ErrInvalidDetails) {
			t.Fatalf("invalid details: %v", err)
		}
	}
}
