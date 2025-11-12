package storage

import "time"

// Game represents a single match
type Game struct {
	GameID    uint      `gorm:"primaryKey"`
	GameType  string    //which game - tic-tac-toe or any new snake
	WinnerID  string    //username, or temporary session ID
	CreatedAt time.Time `gorm:"autoCreateTime"`
	Scores    []Score   `gorm:"foreignKey:GameID"` //will look into score table , which will give list of scores
}

// Score represents player scores linked to a game
type Score struct {
	ScoreID  uint `gorm:"primaryKey"`
	GameID   uint `gorm:"index"`
	PlayerID string
	Score    int
}

// ┌───────────────┐
// │    games      │
// │---------------│
// │ game_id (PK)  │
// │ game_type     │
// │ winner_id     │
// │ created_at    │
// └───────┬───────┘
//         │ 1-to-many
//         │
// ┌───────▼────────┐
// │    scores      │
// │----------------│
// │ score_id (PK)  │
// │ game_id (FK)   │ ← references games.game_id
// │ player_id      │
// │ score          │
// └────────────────┘
