package idgen

type Generator interface {
	Generate() string
}

type Func func() string

func (f Func) Generate() string {
	if f == nil {
		panic("idgen: nil generator func")
	}

	return f()
}
