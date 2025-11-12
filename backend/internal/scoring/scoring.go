package scoring

import "time"

// ScoringEngine implements the time-based scoring rules described in your PDF.
type ScoringEngine struct {
	BasePoints     int
	WinBonus       int
	TimeoutPenalty int
	GracePenalty   int
	MaxMoveTime    time.Duration // e.g. 5s
	GracePeriod    time.Duration // e.g. 3s
}

// New returns a default scoring engine — tweak constants as needed.
func New() *ScoringEngine {
	return &ScoringEngine{
		BasePoints:     5,
		WinBonus:       10,
		TimeoutPenalty: -3,
		GracePenalty:   -5,
		MaxMoveTime:    5 * time.Second,
		GracePeriod:    3 * time.Second,
	}
}

// ScoreMove returns the delta score for a move based on elapsed time and flags.
// timedOut indicates player exceeded MaxMoveTime; usedGrace indicates player used grace window.
func (s *ScoringEngine) ScoreMove(elapsed time.Duration, timedOut bool, usedGrace bool) int {
	if timedOut {
		if usedGrace {
			return s.GracePenalty
		}
		return s.TimeoutPenalty
	}
	// Bonus = (MaxMoveTime - elapsed) clamped to [0, MaxMoveTime], converted to seconds integer
	bonusSec := int((s.MaxMoveTime - elapsed).Round(time.Second) / time.Second)
	if bonusSec < 0 {
		bonusSec = 0
	}
	return s.BasePoints + bonusSec
}

// ScoreWin returns the bonus to apply when a player wins.
func (s *ScoringEngine) ScoreWin() int {
	return s.WinBonus
}
