package password

type Verifier interface {
	Verify(password, encodedHash string) (bool, error)
}
