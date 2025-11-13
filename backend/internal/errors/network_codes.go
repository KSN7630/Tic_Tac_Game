package errors

const (
	ErrInvalidMessage ErrorCode = "invalid_message"
	ErrBadJSON        ErrorCode = "bad_json"
	ErrUnknownAction  ErrorCode = "unknown_action"
	NotRecorded       ErrorCode = "no_moves_recorded"
	ErrInvalidRequest ErrorCode = "invalid_request"
	ErrInvalidUser    ErrorCode = "Invalid_user_or_room"
	ErrUnknown        ErrorCode = "unknown_error"
)
