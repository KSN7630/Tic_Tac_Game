package errors

const (
	// Redis read/write failures
	ErrRedisWriteFailed ErrorCode = "redis_write_failed"
	ErrRedisReadFailed  ErrorCode = "redis_read_failed"

	// Returned when the requested key does not exist in Redis
	ErrNotFound ErrorCode = "redis_key_not_found"

	// JSON marshal/unmarshal errors
	ErrDataEncodingFailed ErrorCode = "data_encoding_failed"
	ErrDataDecodingFailed ErrorCode = "data_decoding_failed"

	// Room/Board/Chat specific keys missing or corrupted
	ErrInvalidRedisData ErrorCode = "invalid_redis_data"

	// Used when Redis connection itself fails
	ErrRedisConnection ErrorCode = "redis_connection_failed"
)
