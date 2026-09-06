package apihttp

const (
	ErrorCodeFileTooLarge         = "FILE_TOO_LARGE"
	ErrorCodeRateLimited          = "RATE_LIMITED"
	ErrorCodeRateLimitUnavailable = "RATE_LIMIT_UNAVAILABLE"

	RateLimitedMessage          = "You can upload up to 10 files per hour. Please try again later."
	DailyAllowanceMessage       = "Your 24-hour upload allowance has been reached. Please try again later."
	RateLimitUnavailableMessage = "The upload quota service is temporarily unavailable. Please try again shortly."
)
