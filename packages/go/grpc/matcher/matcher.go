package matcher

import "strings"

type Matcher func(fullMethod string) bool

func MatchAll(string) bool {
	return true
}

func MatchNone(string) bool {
	return false
}

func MatchMethods(methods ...string) Matcher {
	values := make(map[string]struct{}, len(methods))

	for _, method := range methods {
		if method != "" {
			values[method] = struct{}{}
		}
	}

	return func(fullMethod string) bool {
		_, ok := values[fullMethod]
		return ok
	}
}

func MatchServices(services ...string) Matcher {
	values := make(map[string]struct{}, len(services))

	for _, service := range services {
		service = strings.Trim(service, "/")
		if service != "" {
			values[service] = struct{}{}
		}
	}

	return func(fullMethod string) bool {
		service, _ := SplitFullMethod(fullMethod)
		_, ok := values[service]
		return ok
	}
}

func Not(m Matcher) Matcher {
	if m == nil {
		return MatchAll
	}

	return func(fullMethod string) bool {
		return !m(fullMethod)
	}
}

func Any(matchers ...Matcher) Matcher {
	return func(fullMethod string) bool {
		for _, m := range matchers {
			if m != nil && m(fullMethod) {
				return true
			}
		}

		return false
	}
}

func All(matchers ...Matcher) Matcher {
	return func(fullMethod string) bool {
		for _, m := range matchers {
			if m != nil && !m(fullMethod) {
				return false
			}
		}

		return true
	}
}

func Matches(m Matcher, fullMethod string) bool {
	return m == nil || m(fullMethod)
}

func SplitFullMethod(fullMethod string) (string, string) {
	value := strings.TrimPrefix(fullMethod, "/")

	index := strings.LastIndexByte(value, '/')
	if index < 0 {
		return value, ""
	}

	return value[:index], value[index+1:]
}
