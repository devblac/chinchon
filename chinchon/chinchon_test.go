package chinchon

import (
	"testing"
)

func TestNewGameState(t *testing.T) {
	gs := New()

	if gs == nil {
		t.Fatal("GameState should not be nil")
	}

	if gs.RoundNumber != 1 {
		t.Errorf("Expected round number 1, got %d", gs.RoundNumber)
	}

	if len(gs.Players) != 2 {
		t.Errorf("Expected 2 players, got %d", len(gs.Players))
	}

	// Check that each player has 7 cards
	for playerID, player := range gs.Players {
		if len(player.Hand.Cards) != 7 {
			t.Errorf("Player %d should have 7 cards, got %d", playerID, len(player.Hand.Cards))
		}
	}

	// Check that discard pile has 1 card
	if len(gs.DiscardPile) != 1 {
		t.Errorf("Discard pile should have 1 card, got %d", len(gs.DiscardPile))
	}

	// Check that game is not ended
	if gs.IsGameEnded {
		t.Error("Game should not be ended at start")
	}
}

func TestCardPenaltyValue(t *testing.T) {
	tests := []struct {
		card     Card
		expected int
	}{
		{Card{Suit: ORO, Number: 1}, 1},
		{Card{Suit: COPA, Number: 5}, 5},
		{Card{Suit: ESPADA, Number: 10}, 10},
		{Card{Suit: BASTO, Number: 11}, 10},
		{Card{Suit: ORO, Number: 12}, 10},
	}

	for _, test := range tests {
		actual := test.card.PenaltyValue()
		if actual != test.expected {
			t.Errorf("Card %v should have penalty value %d, got %d",
				test.card, test.expected, actual)
		}
	}
}

func TestHandValidGroups(t *testing.T) {
	// Test a run
	hand := Hand{
		Cards: []Card{
			{Suit: ORO, Number: 1},
			{Suit: ORO, Number: 2},
			{Suit: ORO, Number: 3},
			{Suit: COPA, Number: 5},
		},
	}

	groups := hand.ValidGroups()
	if len(groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(groups))
	}

	if len(groups[0]) != 3 {
		t.Errorf("Expected group of 3 cards, got %d", len(groups[0]))
	}
}

func TestHandIsChinchon(t *testing.T) {
	// Test a Chinchón (7 consecutive cards of same suit)
	hand := Hand{
		Cards: []Card{
			{Suit: ORO, Number: 1},
			{Suit: ORO, Number: 2},
			{Suit: ORO, Number: 3},
			{Suit: ORO, Number: 4},
			{Suit: ORO, Number: 5},
			{Suit: ORO, Number: 6},
			{Suit: ORO, Number: 7},
		},
	}

	if !hand.IsChinchon() {
		t.Error("Hand should be a Chinchón")
	}

	// Test a non-Chinchón
	hand2 := Hand{
		Cards: []Card{
			{Suit: ORO, Number: 1},
			{Suit: COPA, Number: 2},
			{Suit: ORO, Number: 3},
			{Suit: ORO, Number: 4},
			{Suit: ORO, Number: 5},
			{Suit: ORO, Number: 6},
			{Suit: ORO, Number: 7},
		},
	}

	if hand2.IsChinchon() {
		t.Error("Hand should not be a Chinchón")
	}
}

func TestBasicGameFlow(t *testing.T) {
	gs := New()

	// Test drawing from deck
	drawAction := NewActionDrawFromDeck(gs.TurnPlayerID)
	if !drawAction.IsPossible(*gs) {
		t.Error("Drawing from deck should be possible at start")
	}

	err := gs.RunAction(drawAction)
	if err != nil {
		t.Errorf("Error running draw action: %v", err)
	}

	// Player should now have 8 cards and HasDrawnCard should be true
	if len(gs.Players[gs.TurnPlayerID].Hand.Cards) != 8 {
		t.Errorf("Player should have 8 cards after drawing, got %d",
			len(gs.Players[gs.TurnPlayerID].Hand.Cards))
	}

	if !gs.HasDrawnCard {
		t.Error("HasDrawnCard should be true after drawing")
	}

	// Test discarding a card
	cardToDiscard := gs.Players[gs.TurnPlayerID].Hand.Cards[0]
	discardAction := NewActionDiscardCard(cardToDiscard, gs.TurnPlayerID)

	if !discardAction.IsPossible(*gs) {
		t.Error("Discarding should be possible after drawing")
	}

	err = gs.RunAction(discardAction)
	if err != nil {
		t.Errorf("Error running discard action: %v", err)
	}

	// Player should now have 7 cards and turn should have switched
	if len(gs.Players[0].Hand.Cards) != 7 {
		t.Errorf("Player should have 7 cards after discarding, got %d",
			len(gs.Players[0].Hand.Cards))
	}

	// Turn should have switched
	if gs.TurnPlayerID == 0 {
		t.Error("Turn should have switched after discard")
	}
}
