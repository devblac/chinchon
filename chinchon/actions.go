package chinchon

import (
	"fmt"
	"strings"
)

// Action names constants
const (
	DRAW_FROM_DECK         = "draw_from_deck"
	DRAW_FROM_DISCARD      = "draw_from_discard"
	DISCARD_CARD           = "discard_card"
	CLOSE_ROUND            = "close_round"
	CONFIRM_ROUND_FINISHED = "confirm_round_finished"
)

type act struct {
	Name     string `json:"name"`
	PlayerID int    `json:"playerID"`

	fmt.Stringer `json:"-"`
}

func (a act) GetName() string {
	return a.Name
}

func (a act) GetPlayerID() int {
	return a.PlayerID
}

func (a act) GetPriority() int {
	return 0
}

func (a act) AllowLowerPriority() bool {
	return false
}

// By default, actions don't need to be enriched.
func (a act) Enrich(g GameState) {}

func (a act) String() string {
	name := strings.ReplaceAll(strings.TrimPrefix(a.Name, ""), "_", " ")
	return fmt.Sprintf("Player %v %v", a.PlayerID, name)
}

func (a act) YieldsTurn(g GameState) bool {
	return true
}

// ActionDrawFromDeck represents drawing a card from the deck
type ActionDrawFromDeck struct {
	act
}

func NewActionDrawFromDeck(playerID int) Action {
	return &ActionDrawFromDeck{act: act{Name: DRAW_FROM_DECK, PlayerID: playerID}}
}

func (a ActionDrawFromDeck) IsPossible(g GameState) bool {
	if g.IsRoundFinished || g.IsGameEnded {
		return false
	}
	if a.PlayerID != g.TurnPlayerID {
		return false
	}
	if g.HasDrawnCard {
		return false // Already drawn this turn
	}
	return !g.DrawPile.isEmpty()
}

func (a ActionDrawFromDeck) Run(g *GameState) error {
	if !a.IsPossible(*g) {
		return errActionNotPossible
	}

	card, err := g.DrawPile.drawCard()
	if err != nil {
		return err
	}

	g.Players[a.PlayerID].Hand.AddCard(card)
	g.HasDrawnCard = true

	return nil
}

func (a ActionDrawFromDeck) YieldsTurn(g GameState) bool {
	return false // Player must discard after drawing
}

func (a ActionDrawFromDeck) String() string {
	return fmt.Sprintf("Player %v draws from deck", a.PlayerID)
}

// ActionDrawFromDiscard represents drawing a card from the discard pile
type ActionDrawFromDiscard struct {
	act
}

func NewActionDrawFromDiscard(playerID int) Action {
	return &ActionDrawFromDiscard{act: act{Name: DRAW_FROM_DISCARD, PlayerID: playerID}}
}

func (a ActionDrawFromDiscard) IsPossible(g GameState) bool {
	if g.IsRoundFinished || g.IsGameEnded {
		return false
	}
	if a.PlayerID != g.TurnPlayerID {
		return false
	}
	if g.HasDrawnCard {
		return false // Already drawn this turn
	}
	return len(g.DiscardPile) > 0
}

func (a ActionDrawFromDiscard) Run(g *GameState) error {
	if !a.IsPossible(*g) {
		return errActionNotPossible
	}

	if len(g.DiscardPile) == 0 {
		return fmt.Errorf("discard pile is empty")
	}

	// Take the top card from discard pile
	card := g.DiscardPile[len(g.DiscardPile)-1]
	g.DiscardPile = g.DiscardPile[:len(g.DiscardPile)-1]

	g.Players[a.PlayerID].Hand.AddCard(card)
	g.HasDrawnCard = true

	return nil
}

func (a ActionDrawFromDiscard) YieldsTurn(g GameState) bool {
	return false // Player must discard after drawing
}

func (a ActionDrawFromDiscard) String() string {
	return fmt.Sprintf("Player %v draws from discard pile", a.PlayerID)
}

// ActionDiscardCard represents discarding a card
type ActionDiscardCard struct {
	act
	Card Card `json:"card"`
}

func NewActionDiscardCard(card Card, playerID int) Action {
	return &ActionDiscardCard{
		act:  act{Name: DISCARD_CARD, PlayerID: playerID},
		Card: card,
	}
}

func (a ActionDiscardCard) IsPossible(g GameState) bool {
	if g.IsRoundFinished || g.IsGameEnded {
		return false
	}
	if a.PlayerID != g.TurnPlayerID {
		return false
	}
	if !g.HasDrawnCard {
		return false // Must draw before discarding
	}

	// Check if player has the card
	return g.Players[a.PlayerID].Hand.HasCard(a.Card)
}

func (a ActionDiscardCard) Run(g *GameState) error {
	if !a.IsPossible(*g) {
		return errActionNotPossible
	}

	err := g.Players[a.PlayerID].Hand.RemoveCard(a.Card)
	if err != nil {
		return err
	}

	// Add card to discard pile
	g.DiscardPile = append(g.DiscardPile, a.Card)

	return nil
}

func (a ActionDiscardCard) YieldsTurn(g GameState) bool {
	return true // Turn ends after discarding
}

func (a ActionDiscardCard) String() string {
	return fmt.Sprintf("Player %v discards %v", a.PlayerID, a.Card)
}

// ActionClose represents closing the round
type ActionClose struct {
	act
}

func NewActionClose(playerID int) Action {
	return &ActionClose{act: act{Name: CLOSE_ROUND, PlayerID: playerID}}
}

func (a ActionClose) IsPossible(g GameState) bool {
	if g.IsRoundFinished || g.IsGameEnded {
		return false
	}
	if a.PlayerID != g.TurnPlayerID {
		return false
	}
	if !g.HasDrawnCard {
		return false // Must draw before closing
	}

	return g.CanClose(a.PlayerID)
}

func (a ActionClose) Run(g *GameState) error {
	if !a.IsPossible(*g) {
		return errActionNotPossible
	}

	g.CloseRound(a.PlayerID)

	return nil
}

func (a ActionClose) YieldsTurn(g GameState) bool {
	return true
}

func (a ActionClose) String() string {
	return fmt.Sprintf("Player %v closes the round", a.PlayerID)
}

// ActionConfirmRoundFinished represents confirming that the round is finished
type ActionConfirmRoundFinished struct {
	act
}

func NewActionConfirmRoundFinished(playerID int) Action {
	return &ActionConfirmRoundFinished{act: act{Name: CONFIRM_ROUND_FINISHED, PlayerID: playerID}}
}

func (a ActionConfirmRoundFinished) IsPossible(g GameState) bool {
	if !g.IsRoundFinished || g.IsGameEnded {
		return false
	}

	// Check if this player has already confirmed
	return !g.RoundFinishedConfirmedPlayerIDs[a.PlayerID]
}

func (a ActionConfirmRoundFinished) Run(g *GameState) error {
	if !a.IsPossible(*g) {
		return errActionNotPossible
	}

	g.RoundFinishedConfirmedPlayerIDs[a.PlayerID] = true

	return nil
}

func (a ActionConfirmRoundFinished) YieldsTurn(g GameState) bool {
	return false // Don't switch turns when confirming
}

func (a ActionConfirmRoundFinished) String() string {
	return fmt.Sprintf("Player %v confirms round finished", a.PlayerID)
}
