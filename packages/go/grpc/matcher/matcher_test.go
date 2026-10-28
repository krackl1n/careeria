package matcher_test

import (
	"testing"

	"github.com/krackl1n/careeria/grpc/matcher"

	"github.com/stretchr/testify/assert"
)

const fullMethod = "/toolbox.TestService/Create"

func TestMatchers(t *testing.T) {
	t.Parallel()

	assert.True(t, matcher.MatchAll(fullMethod))
	assert.False(t, matcher.MatchNone(fullMethod))
	assert.True(t, matcher.MatchMethods(fullMethod)(fullMethod))
	assert.False(t, matcher.MatchMethods("/other.Service/Get")(fullMethod))
	assert.True(t, matcher.MatchServices("toolbox.TestService")(fullMethod))
	assert.True(t, matcher.MatchServices("/toolbox.TestService/")(fullMethod))
	assert.False(t, matcher.MatchServices("other.Service")(fullMethod))
}

func TestMatcherComposition(t *testing.T) {
	t.Parallel()

	assert.True(t, matcher.Matches(nil, fullMethod))
	assert.True(t, matcher.Not(nil)(fullMethod))
	assert.True(t, matcher.Not(matcher.MatchMethods("/other.Service/Get"))(fullMethod))
	assert.True(t, matcher.Any(nil, matcher.MatchMethods(fullMethod))(fullMethod))
	assert.False(t, matcher.Any(nil, matcher.MatchMethods("/other.Service/Get"))(fullMethod))
	assert.True(t, matcher.All(nil, matcher.MatchServices("toolbox.TestService"))(fullMethod))
	assert.False(t, matcher.All(matcher.MatchAll, matcher.MatchMethods("/other.Service/Get"))(fullMethod))
}

func TestSplitFullMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fullMethod string
		service    string
		method     string
	}{
		{
			name:       "service and method",
			fullMethod: "/toolbox.TestService/Create",
			service:    "toolbox.TestService",
			method:     "Create",
		},
		{
			name:       "without leading slash",
			fullMethod: "toolbox.TestService/Create",
			service:    "toolbox.TestService",
			method:     "Create",
		},
		{
			name:       "service only",
			fullMethod: "toolbox.TestService",
			service:    "toolbox.TestService",
		},
		{
			name:       "empty",
			fullMethod: "",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, method := matcher.SplitFullMethod(tt.fullMethod)

			assert.Equal(t, tt.service, service)
			assert.Equal(t, tt.method, method)
		})
	}
}
