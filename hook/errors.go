package hook

import "errors"

var (
	ErrHookNotFound            = errors.New("hook not found")
	ErrHookAlreadyExists       = errors.New("hook already exists")
	ErrEmptyHookID             = errors.New("hook id cannot be empty")
	ErrRateLimitExceeded       = errors.New("rate limit exceeded")
	ErrGlobalRateLimitExceeded = errors.New("global rate limit exceeded")
	ErrTopicRateLimitExceeded  = errors.New("topic rate limit exceeded")
)
