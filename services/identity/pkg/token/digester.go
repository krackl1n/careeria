package token

type Digester interface {
	Digest(token string) string
}
