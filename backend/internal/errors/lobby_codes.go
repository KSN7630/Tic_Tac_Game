package errors

const (
	ErrRoomNotFound     ErrorCode = "room_not_found"
	ErrChallengeCreate  ErrorCode = "challenge_create_failed"
	ErrChallengeAccept  ErrorCode = "challenge_accept_failed"
	ErrRoomFull         ErrorCode = "room_full"
	ErrRoomCreateFailed ErrorCode = "room_create_failed"
	ErrBroadcastFailed  ErrorCode = "broadcast_failed"
)
