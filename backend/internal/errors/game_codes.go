package errors

// Error codes used by GameEngine (engine.go logic)
const (
	// Board validation
	ErrNilBoard     ErrorCode = "nil_board"
	ErrGameFinished ErrorCode = "game_finished"

	// Move validation
	ErrIndexOutOfRange ErrorCode = "index_out_of_range"
	ErrCellOccupied    ErrorCode = "cell_occupied"
	ErrNotPlayersTurn  ErrorCode = "not_players_turn"

	// Player/mark issues
	ErrUnknownPlayerMark ErrorCode = "unknown_player_mark"

	// Timing / rulebook
	ErrMoveTimeout  ErrorCode = "move_timeout"
	ErrGraceTimeout ErrorCode = "grace_timeout" // >5s and >10s rule
	ErrTotalTimeout ErrorCode = "total_timeout" // full exceeded

	// Game state issues
	ErrInvalidBoardState ErrorCode = "invalid_board_state"
)
