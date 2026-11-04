package authentication

type Method string

const (
	MethodPassword     Method = "password"
	MethodTOTP         Method = "totp"
	MethodWebAuthn     Method = "webauthn"
	MethodRecoveryCode Method = "recovery_code"
)

func (m Method) IsValid() bool {
	switch m {
	case MethodPassword, MethodTOTP, MethodWebAuthn, MethodRecoveryCode:
		return true
	default:
		return false
	}
}

type Level string

const (
	LevelSingleFactor Level = "single_factor"
	LevelMultiFactor  Level = "multi_factor"
)

func (l Level) IsValid() bool {
	return l == LevelSingleFactor || l == LevelMultiFactor
}
