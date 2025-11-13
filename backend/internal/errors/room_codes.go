package errors

// Error codes used inside room.go

const (
	// Adding players

	ErrPlayerAlreadyIn ErrorCode = "player_already_in_room"

	// Starting game
	ErrNotEnoughPlayers ErrorCode = "not_enough_players"
	ErrSaveBoardFailed  ErrorCode = "save_board_failed"

	// Move errors (room-level, NOT engine-level)
	ErrBoardNotLoaded ErrorCode = "board_not_loaded"

	// Replay / Redis issues
	ErrReplaySaveFailed ErrorCode = "replay_save_failed"
	ErrBoardSaveFailed  ErrorCode = "board_save_failed"
	ErrRoomSaveFailed   ErrorCode = "room_save_failed"
)
