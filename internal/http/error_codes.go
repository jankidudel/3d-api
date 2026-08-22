package apihttp

const (
	ErrorCodeFileTooLarge         = "FILE_TOO_LARGE"
	ErrorCodeRateLimited          = "RATE_LIMITED"
	ErrorCodeRateLimitUnavailable = "RATE_LIMIT_UNAVAILABLE"

	RateLimitedMessage          = "You have reached the free upload limit. Please try again later."
	RateLimitUnavailableMessage = "The upload quota service is temporarily unavailable. Please try again shortly."
)
