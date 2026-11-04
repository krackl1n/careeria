package securityevent

const (
	EventTypeAuthorizationCodeExchange EventType = "OAuthAuthorizationCodeExchange"
	EventTypeRefreshTokenExchange      EventType = "OAuthRefreshTokenExchange"
	EventTypeTokenRevocation           EventType = "OAuthTokenRevocation"
	EventTypeUserInfoAccess            EventType = "OAuthUserInfoAccess"
)
