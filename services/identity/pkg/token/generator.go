package token

type Generator interface {
	Generate() (string, error)
}
