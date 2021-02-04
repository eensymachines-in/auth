package auth

// ErrQueryFailed : when the mongo query, or redis query fails
type ErrQueryFailed error

// ErrDuplicate : this is when duplicate insertion of any resource
type ErrDuplicate error

// ErrNotFound : this is when no result is fetched and atleast one was expected
type ErrNotFound error

// ErrInvalid : this is when one or more fields are invalid and cannot proceed with query
type ErrInvalid error

// ErrUnauth : this is when the action is not allowed
type ErrUnauth error

// ErrCache : anytime we have a problem setting or getting from cache
type ErrCache error

// ErrTokExpired : this is when no record in the cache found with auth uuid
type ErrTokExpired error

// ErrEncrypt : this is when one or more hashing algorithms fail
type ErrEncrypt error
