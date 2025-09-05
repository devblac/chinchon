package chinchon

import (
	"errors"
	"fmt"
	"math/rand"
)

const (
	ORO    = "oro"
	COPA   = "copa"
	ESPADA = "espada"
	BASTO  = "basto"
)

// Card represents a Spanish deck card.
type Card struct {
	// Suit is the card's suit, which can be "oro", "copa", "espada" or "basto".
	Suit string `json:"suit"`

	// Number is the card's number, from 1 to 12 (including 8 and 9 for Chinchón).
	Number int `json:"number"`
}

func (c Card) String() string {
	return fmt.Sprintf("%d de %s", c.Number, c.Suit)
}

// PenaltyValue returns the penalty points for this card in Chinchón scoring
func (c Card) PenaltyValue() int {
	if c.Number >= 10 {
		return 10 // Face cards (10, 11, 12) are worth 10 points
	}
	return c.Number // Number cards are worth their face value
}

type deck struct {
	cards        []Card
	dealHandFunc func() *Hand
}

// Hand represents a player's hand in Chinchón. Players have 7 cards.
type Hand struct {
	Cards []Card `json:"cards"`
}

func (h Hand) DeepCopy() Hand {
	cpyCards := make([]Card, len(h.Cards))
	copy(cpyCards, h.Cards)
	return Hand{Cards: cpyCards}
}

func (h Hand) HasCard(c Card) bool {
	for _, card := range h.Cards {
		if card == c {
			return true
		}
	}
	return false
}

func (h *Hand) AddCard(card Card) {
	h.Cards = append(h.Cards, card)
}

func (h *Hand) RemoveCard(card Card) error {
	for i, c := range h.Cards {
		if c == card {
			h.Cards = append(h.Cards[:i], h.Cards[i+1:]...)
			return nil
		}
	}
	return errCardNotInHand
}

// ValidGroups returns all valid runs and sets in the hand
func (h Hand) ValidGroups() [][]Card {
	var groups [][]Card

	// Find runs (consecutive cards of same suit)
	runs := h.findRuns()
	groups = append(groups, runs...)

	// Find sets (same number, different suits)
	sets := h.findSets()
	groups = append(groups, sets...)

	return groups
}

// findRuns finds all valid runs (3+ consecutive cards of same suit)
func (h Hand) findRuns() [][]Card {
	var runs [][]Card
	suitCards := make(map[string][]Card)

	// Group cards by suit
	for _, card := range h.Cards {
		suitCards[card.Suit] = append(suitCards[card.Suit], card)
	}

	// For each suit, find consecutive runs
	for _, cards := range suitCards {
		if len(cards) < 3 {
			continue
		}

		// Sort cards by number
		for i := 0; i < len(cards)-1; i++ {
			for j := i + 1; j < len(cards); j++ {
				if cards[i].Number > cards[j].Number {
					cards[i], cards[j] = cards[j], cards[i]
				}
			}
		}

		// Find consecutive sequences
		for i := 0; i <= len(cards)-3; i++ {
			run := []Card{cards[i]}
			for j := i + 1; j < len(cards); j++ {
				if cards[j].Number == run[len(run)-1].Number+1 {
					run = append(run, cards[j])
				} else {
					break
				}
			}
			if len(run) >= 3 {
				runs = append(runs, run)
			}
		}
	}

	return runs
}

// findSets finds all valid sets (3 or 4 cards of same number, different suits)
func (h Hand) findSets() [][]Card {
	var sets [][]Card
	numberCards := make(map[int][]Card)

	// Group cards by number
	for _, card := range h.Cards {
		numberCards[card.Number] = append(numberCards[card.Number], card)
	}

	// Find sets of 3 or 4 cards with same number
	for _, cards := range numberCards {
		if len(cards) >= 3 {
			// Check if all cards have different suits
			suits := make(map[string]bool)
			validSet := true
			for _, card := range cards {
				if suits[card.Suit] {
					validSet = false
					break
				}
				suits[card.Suit] = true
			}
			if validSet {
				sets = append(sets, cards)
			}
		}
	}

	return sets
}

// IsChinchon returns true if the hand contains a Chinchón (7 consecutive cards of same suit)
func (h Hand) IsChinchon() bool {
	if len(h.Cards) != 7 {
		return false
	}

	suitCards := make(map[string][]Card)
	for _, card := range h.Cards {
		suitCards[card.Suit] = append(suitCards[card.Suit], card)
	}

	// Check if all 7 cards are of the same suit
	for _, cards := range suitCards {
		if len(cards) == 7 {
			// Sort cards by number
			for i := 0; i < len(cards)-1; i++ {
				for j := i + 1; j < len(cards); j++ {
					if cards[i].Number > cards[j].Number {
						cards[i], cards[j] = cards[j], cards[i]
					}
				}
			}

			// Check if consecutive
			consecutive := true
			for i := 1; i < len(cards); i++ {
				if cards[i].Number != cards[i-1].Number+1 {
					consecutive = false
					break
				}
			}
			return consecutive
		}
	}

	return false
}

// PenaltyPoints calculates penalty points for ungrouped cards
func (h Hand) PenaltyPoints(groups [][]Card) int {
	// Create a map of grouped cards
	groupedCards := make(map[Card]bool)
	for _, group := range groups {
		for _, card := range group {
			groupedCards[card] = true
		}
	}

	penalty := 0
	for _, card := range h.Cards {
		if !groupedCards[card] {
			penalty += card.PenaltyValue()
		}
	}

	return penalty
}

var (
	errCardNotInHand = errors.New("card not in hand")
)

// makeSpanishCards creates a full 40-card Spanish deck (including 8s and 9s)
func makeSpanishCards() []Card {
	cards := []Card{}
	suits := []string{ORO, COPA, ESPADA, BASTO}
	for _, suit := range suits {
		for i := 1; i <= 12; i++ {
			// Include all cards from 1 to 12 for Chinchón (including 8 and 9)
			cards = append(cards, Card{Suit: suit, Number: i})
		}
	}

	rand.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})

	return cards
}

func newDeck() *deck {
	d := deck{cards: makeSpanishCards()}
	d.dealHandFunc = d.defaultDealHand
	return &d
}

func (d *deck) shuffle() {
	d.cards = makeSpanishCards()
}

func (d *deck) dealHand() *Hand {
	return d.dealHandFunc()
}

// defaultDealHand deals 7 cards for Chinchón
func (d *deck) defaultDealHand() *Hand {
	hand := &Hand{}
	for i := 0; i < 7; i++ {
		if len(d.cards) > 0 {
			hand.Cards = append(hand.Cards, d.cards[0])
			d.cards = d.cards[1:]
		}
	}
	return hand
}

// drawCard draws the top card from the deck
func (d *deck) drawCard() (Card, error) {
	if len(d.cards) == 0 {
		return Card{}, errors.New("deck is empty")
	}
	card := d.cards[0]
	d.cards = d.cards[1:]
	return card, nil
}

// isEmpty returns true if the deck has no cards left
func (d *deck) isEmpty() bool {
	return len(d.cards) == 0
}

// remainingCards returns the number of cards left in the deck
func (d *deck) remainingCards() int {
	return len(d.cards)
}
