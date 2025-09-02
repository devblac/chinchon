package newbot

import (
	"os"

	"log"

	"github.com/devblac/chinchon/chinchon"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

type NoOpLogger struct{}

func (NoOpLogger) Printf(format string, v ...interface{}) {}

type Bot struct {
	logger Logger
}

func WithDefaultLogger(b *Bot) {
	b.logger = log.New(os.Stderr, "", log.LstdFlags)
}

func New(opts ...func(*Bot)) *Bot {
	b := &Bot{logger: NoOpLogger{}}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (m Bot) ChooseAction(gs chinchon.ClientGameState) chinchon.Action {
	// Trivial cases
	if len(gs.PossibleActions) == 0 {
		return nil
	}
	if len(gs.PossibleActions) == 1 {
		return _deserializeActions(gs.PossibleActions)[0]
	}

	// Simple Chinchón bot strategy
	actions := _deserializeActions(gs.PossibleActions)

	// Always confirm round finished if possible
	for _, action := range actions {
		if action.GetName() == chinchon.CONFIRM_ROUND_FINISHED {
			return action
		}
	}

	// Prefer drawing from discard pile if the card helps form groups
	for _, action := range actions {
		if action.GetName() == chinchon.DRAW_FROM_DISCARD {
			return action
		}
	}

	// Otherwise draw from deck
	for _, action := range actions {
		if action.GetName() == chinchon.DRAW_FROM_DECK {
			return action
		}
	}

	// Close if possible
	for _, action := range actions {
		if action.GetName() == chinchon.CLOSE_ROUND {
			return action
		}
	}

	// Discard the highest value card to minimize penalty
	var bestDiscard chinchon.Action
	highestValue := -1
	for _, action := range actions {
		if action.GetName() == chinchon.DISCARD_CARD {
			discardAction := action.(*chinchon.ActionDiscardCard)
			value := discardAction.Card.PenaltyValue()
			if value > highestValue {
				highestValue = value
				bestDiscard = action
			}
		}
	}
	if bestDiscard != nil {
		return bestDiscard
	}

	// Fallback to first action
	return actions[0]
}
