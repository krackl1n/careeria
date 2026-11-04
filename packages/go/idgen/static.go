package idgen

type Static struct {
	value string
}

func NewStatic(value string) Static {
	if value == "" {
		panic("idgen: empty static value")
	}

	return Static{
		value: value,
	}
}

func (s Static) Generate() string {
	if s.value == "" {
		panic("idgen: empty static value")
	}

	return s.value
}
