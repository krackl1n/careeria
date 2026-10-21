package authentication

import "errors"

type factorClass uint8

const (
	factorClassKnowledge factorClass = iota + 1
	factorClassPossession
)

func ResolveLevel(methods []Method) (Level, error) {
	if len(methods) == 0 {
		return "", errors.New("resolve authentication level: at least one method is required")
	}

	classes := make(map[factorClass]struct{}, 2)
	seen := make(map[Method]struct{}, len(methods))
	for _, method := range methods {
		if _, exists := seen[method]; exists {
			return "", errors.New("resolve authentication level: method is duplicated")
		}
		seen[method] = struct{}{}

		class, ok := method.factorClass()
		if !ok {
			return "", errors.New("resolve authentication level: unknown method")
		}
		classes[class] = struct{}{}
	}

	if len(classes) > 1 {
		return LevelMultiFactor, nil
	}
	return LevelSingleFactor, nil
}

func (m Method) factorClass() (factorClass, bool) {
	switch m {
	case MethodPassword:
		return factorClassKnowledge, true
	case MethodTOTP, MethodWebAuthn, MethodRecoveryCode:
		return factorClassPossession, true
	default:
		return 0, false
	}
}
