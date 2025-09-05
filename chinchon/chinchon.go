package chinchon

import (
	"encoding/json"
	"errors"
	"fmt"
)

// DefaultMaxPoints is the points a player must reach to lose the game.
const DefaultMaxPoints = 100

// GameState represents the state of a Chinchón game.
type GameState struct {
	// RoundNumber is the number of the current round, starting from 1.
	RoundNumber int `json:"roundNumber"`

	// TurnPlayerID is the player ID of the player whose turn it is to play an action.
	TurnPlayerID int `json:"turnPlayerID"`

	// TurnOpponentPlayerID is the player ID of the opponent of the player whose turn it is.
	TurnOpponentPlayerID int `json:"turnOpponentPlayerID"`

	// Players is a map of player IDs to their respective hands and scores.
	Players map[int]*Player `json:"players"`

	// DrawPile is the deck of cards to draw from
	DrawPile *deck `json:"-"`

	// DiscardPile is the pile of discarded cards (face up)
	DiscardPile []Card `json:"discardPile"`

	// PossibleActions is a list of possible actions that the current player can take.
	PossibleActions []json.RawMessage `json:"possibleActions"`

	// IsRoundFinished is true if the current round is finished.
	IsRoundFinished bool `json:"isRoundFinished"`

	// IsGameEnded is true if the whole game is ended.
	IsGameEnded bool `json:"isGameEnded"`

	// WinnerPlayerID is the player ID of the player who won the game.
	WinnerPlayerID int `json:"winnerPlayerID"`

	// LoserPlayerID is the player ID of the player who lost the game (reached max points).
	LoserPlayerID int `json:"loserPlayerID"`

	// RoundsLog is the ordered list of logs of each round that was played in the game.
	RoundsLog []*RoundLog `json:"roundsLog"`

	// RoundFinishedConfirmedPlayerIDs tracks which players have confirmed the round is finished
	RoundFinishedConfirmedPlayerIDs map[int]bool `json:"roundFinishedConfirmedPlayerIDs"`

	// RuleMaxPoints is the maximum points before a player loses
	RuleMaxPoints int `json:"ruleMaxPoints"`

	// CurrentRoundClosedByPlayerID is the player who closed the current round, -1 if none
	CurrentRoundClosedByPlayerID int `json:"currentRoundClosedByPlayerID"`

	// HasDrawnCard indicates if the current player has drawn a card this turn
	HasDrawnCard bool `json:"hasDrawnCard"`
}

type Player struct {
	// Hand contains the cards of the player.
	Hand *Hand `json:"hand"`

	// Score is the player's penalty score (from 0 to MaxPoints).
	Score int `json:"score"`
}

// RoundLog is a log of a round that was played in the game
type RoundLog struct {
	// HandsDealt is a map from PlayerID to its hand during this round.
	HandsDealt map[int]*Hand `json:"handsDealt"`

	// WinnerPlayerID is the player who won this round (had fewer penalty points)
	WinnerPlayerID int `json:"winnerPlayerID"`

	// LoserPlayerID is the player who lost this round (had more penalty points)
	LoserPlayerID int `json:"loserPlayerID"`

	// PenaltyPoints is a map of penalty points awarded to each player
	PenaltyPoints map[int]int `json:"penaltyPoints"`

	// ClosedByPlayerID is the player who closed the round, -1 if none
	ClosedByPlayerID int `json:"closedByPlayerID"`

	// WasChinchon indicates if the round was won with a Chinchón
	WasChinchon bool `json:"wasChinchon"`

	// ActionsLog is the ordered list of actions of this round.
	ActionsLog []ActionLog `json:"actionsLog"`
}

// ActionLog is a log of an action that was run in a round.
type ActionLog struct {
	// PlayerID is the player ID of the player who ran the action.
	PlayerID int `json:"playerID"`

	// Action is a JSON-serialized action.
	Action json.RawMessage `json:"action"`
}

// WithMaxPoints sets the maximum points required to lose the game.
func WithMaxPoints(maxPoints int) func(*GameState) {
	return func(gs *GameState) {
		gs.RuleMaxPoints = maxPoints
	}
}

func New(opts ...func(*GameState)) *GameState {
	gs := &GameState{
		RoundNumber: 0,
		Players: map[int]*Player{
			0: {Hand: nil, Score: 0},
			1: {Hand: nil, Score: 0},
		},
		DrawPile:                        newDeck(),
		DiscardPile:                     []Card{},
		IsGameEnded:                     false,
		WinnerPlayerID:                  -1,
		LoserPlayerID:                   -1,
		RoundsLog:                       []*RoundLog{{}}, // initialised with an empty round to be 1-indexed
		RuleMaxPoints:                   DefaultMaxPoints,
		CurrentRoundClosedByPlayerID:    -1,
		RoundFinishedConfirmedPlayerIDs: map[int]bool{},
		HasDrawnCard:                    false,
	}

	for _, opt := range opts {
		opt(gs)
	}

	gs.startNewRound()

	return gs
}

func (g *GameState) startNewRound() {
	g.DrawPile.shuffle()
	g.RoundNumber++

	// Alternate who starts each round
	if g.RoundNumber == 1 {
		g.TurnPlayerID = 0
	} else {
		g.TurnPlayerID = g.OpponentOf(g.TurnPlayerID)
	}
	g.TurnOpponentPlayerID = g.OpponentOf(g.TurnPlayerID)

	// Deal 7 cards to each player
	g.Players[0].Hand = g.DrawPile.dealHand()
	g.Players[1].Hand = g.DrawPile.dealHand()

	// Place one card face up to start the discard pile
	if !g.DrawPile.isEmpty() {
		topCard, _ := g.DrawPile.drawCard()
		g.DiscardPile = []Card{topCard}
	}

	g.IsRoundFinished = false
	g.CurrentRoundClosedByPlayerID = -1
	g.RoundFinishedConfirmedPlayerIDs = map[int]bool{}
	g.HasDrawnCard = false

	handsDealt := make(map[int]*Hand)
	for playerID, player := range g.Players {
		handCopy := player.Hand.DeepCopy()
		handsDealt[playerID] = &handCopy
	}

	g.RoundsLog = append(g.RoundsLog, &RoundLog{
		HandsDealt:       handsDealt,
		WinnerPlayerID:   -1,
		LoserPlayerID:    -1,
		PenaltyPoints:    map[int]int{},
		ClosedByPlayerID: -1,
		WasChinchon:      false,
		ActionsLog:       []ActionLog{},
	})

	g.PossibleActions = _serializeActions(g.CalculatePossibleActions())
}

func (g *GameState) RunAction(action Action) error {
	if action == nil {
		return nil
	}

	if g.IsGameEnded {
		return fmt.Errorf("%w trying to run [%v]", errGameIsEnded, action)
	}

	if !g.IsRoundFinished && action.GetPlayerID() != g.TurnPlayerID {
		return errNotYourTurn
	}

	if !action.IsPossible(*g) {
		return fmt.Errorf("%w trying to run [%v]", errActionNotPossible, action)
	}

	err := action.Run(g)
	if err != nil {
		return fmt.Errorf("%w trying to run [%v] after checking it was possible", err, action)
	}

	if action.GetName() != CONFIRM_ROUND_FINISHED {
		g.RoundsLog[g.RoundNumber].ActionsLog = append(g.RoundsLog[g.RoundNumber].ActionsLog, ActionLog{
			PlayerID: g.TurnPlayerID,
			Action:   SerializeAction(action),
		})
	}

	// Start new round if current round is finished
	if !g.IsGameEnded && g.IsRoundFinished && len(g.RoundFinishedConfirmedPlayerIDs) == 2 {
		g.startNewRound()
		return nil
	}

	// Switch player turn within current round (unless current action doesn't yield turn)
	if !g.IsGameEnded && !g.IsRoundFinished && action.YieldsTurn(*g) {
		g.TurnPlayerID, g.TurnOpponentPlayerID = g.TurnOpponentPlayerID, g.TurnPlayerID
		g.HasDrawnCard = false // Reset draw state for new turn
	}

	if !g.IsGameEnded && g.IsRoundFinished && len(g.RoundFinishedConfirmedPlayerIDs) == 1 {
		if g.RoundFinishedConfirmedPlayerIDs[g.TurnPlayerID] {
			g.changeTurn()
		}
	}

	// Handle end of game due to score
	for playerID := range g.Players {
		if g.Players[playerID].Score >= g.RuleMaxPoints {
			g.IsGameEnded = true
			g.LoserPlayerID = playerID
			g.WinnerPlayerID = g.OpponentOf(playerID)
		}
	}

	possibleActions := g.CalculatePossibleActions()
	if g.countActionsOfTurnPlayer() == 0 {
		// If the current player has no actions left, it's the opponent's turn.
		g.changeTurn()
		possibleActions = g.CalculatePossibleActions()
	}

	g.PossibleActions = _serializeActions(possibleActions)

	return nil
}

func (g *GameState) changeTurn() {
	g.TurnPlayerID, g.TurnOpponentPlayerID = g.TurnOpponentPlayerID, g.TurnPlayerID
	g.HasDrawnCard = false
}

func (g GameState) countActionsOfTurnPlayer() int {
	count := 0
	for _, a := range g.CalculatePossibleActions() {
		if a.GetPlayerID() == g.TurnPlayerID {
			count++
		}
	}
	return count
}

func (g GameState) OpponentOf(playerID int) int {
	for id := range g.Players {
		if id != playerID {
			return id
		}
	}
	return -1 // Unreachable
}

func (g GameState) Serialize() ([]byte, error) {
	return json.Marshal(g)
}

func (g *GameState) PrettyPrint() (string, error) {
	var prettyJSON []byte
	prettyJSON, err := json.MarshalIndent(g, "", "    ")
	if err != nil {
		return "", err
	}
	return string(prettyJSON), nil
}

// GetTopDiscardCard returns the top card of the discard pile
func (g GameState) GetTopDiscardCard() (Card, error) {
	if len(g.DiscardPile) == 0 {
		return Card{}, errors.New("discard pile is empty")
	}
	return g.DiscardPile[len(g.DiscardPile)-1], nil
}

// CanClose returns true if the current player can close the round
func (g GameState) CanClose(playerID int) bool {
	if g.IsRoundFinished {
		return false
	}

	hand := g.Players[playerID].Hand
	if hand == nil || len(hand.Cards) != 7 {
		return false
	}

	// Check if player can form valid groups with all cards except one (which will be discarded)
	// For simplicity, we'll check if they can group 6 cards (leaving 1 for discard)
	validGroups := hand.ValidGroups()
	groupedCards := make(map[Card]bool)
	for _, group := range validGroups {
		for _, card := range group {
			groupedCards[card] = true
		}
	}

	ungroupedCount := 0
	for _, card := range hand.Cards {
		if !groupedCards[card] {
			ungroupedCount++
		}
	}

	// Can close if at most 1 card is ungrouped (will be discarded)
	return ungroupedCount <= 1
}

// CloseRound closes the current round and calculates scores
func (g *GameState) CloseRound(closingPlayerID int) {
	g.IsRoundFinished = true
	g.CurrentRoundClosedByPlayerID = closingPlayerID

	// Calculate penalty points for each player
	penaltyPoints := make(map[int]int)

	for playerID, player := range g.Players {
		if player.Hand == nil {
			continue
		}

		// Check for Chinchón
		if player.Hand.IsChinchon() {
			// Chinchón ends the game immediately
			g.IsGameEnded = true
			g.WinnerPlayerID = playerID
			g.LoserPlayerID = g.OpponentOf(playerID)
			g.RoundsLog[g.RoundNumber].WasChinchon = true
			return
		}

		// Calculate penalty points based on ungrouped cards
		validGroups := player.Hand.ValidGroups()
		penalty := player.Hand.PenaltyPoints(validGroups)
		penaltyPoints[playerID] = penalty
	}

	// Determine round winner (player with fewer penalty points)
	player0Penalty := penaltyPoints[0]
	player1Penalty := penaltyPoints[1]

	var roundWinner, roundLoser int
	if player0Penalty < player1Penalty {
		roundWinner = 0
		roundLoser = 1
	} else if player1Penalty < player0Penalty {
		roundWinner = 1
		roundLoser = 0
	} else {
		// Tie - both players get their penalty points
		roundWinner = -1
		roundLoser = -1
	}

	// Award penalty points
	if closingPlayerID != -1 && roundWinner == closingPlayerID {
		// Player who closed won - opponent gets penalty points
		opponentID := g.OpponentOf(closingPlayerID)
		g.Players[opponentID].Score += penaltyPoints[opponentID]

		// If closing player grouped all cards perfectly, opponent gets 10 extra points
		if penaltyPoints[closingPlayerID] == 0 {
			g.Players[opponentID].Score += 10
		}
	} else {
		// Normal scoring - everyone gets their penalty points
		for playerID, penalty := range penaltyPoints {
			g.Players[playerID].Score += penalty
		}
	}

	// Update round log
	g.RoundsLog[g.RoundNumber].WinnerPlayerID = roundWinner
	g.RoundsLog[g.RoundNumber].LoserPlayerID = roundLoser
	g.RoundsLog[g.RoundNumber].PenaltyPoints = penaltyPoints
	g.RoundsLog[g.RoundNumber].ClosedByPlayerID = closingPlayerID
}

type Action interface {
	IsPossible(g GameState) bool
	Run(g *GameState) error
	GetName() string
	GetPlayerID() int
	YieldsTurn(g GameState) bool
	Enrich(g GameState)
	GetPriority() int
	AllowLowerPriority() bool
	fmt.Stringer
}

var (
	errActionNotPossible = errors.New("action not possible")
	errGameIsEnded       = errors.New("game is ended")
	errNotYourTurn       = errors.New("not your turn")
)

func (g GameState) CalculatePossibleActions() []Action {
	allActions := []Action{}

	// Add drawing actions (if player hasn't drawn yet)
	if !g.HasDrawnCard && !g.IsRoundFinished {
		allActions = append(allActions,
			NewActionDrawFromDeck(g.TurnPlayerID),
			NewActionDrawFromDiscard(g.TurnPlayerID),
		)
	}

	// Add discarding actions (if player has drawn)
	if g.HasDrawnCard && !g.IsRoundFinished {
		for _, card := range g.Players[g.TurnPlayerID].Hand.Cards {
			allActions = append(allActions, NewActionDiscardCard(card, g.TurnPlayerID))
		}
	}

	// Add close actions (if player can close)
	if g.CanClose(g.TurnPlayerID) && g.HasDrawnCard {
		allActions = append(allActions, NewActionClose(g.TurnPlayerID))
	}

	// Add confirm round finished actions
	allActions = append(allActions,
		NewActionConfirmRoundFinished(g.TurnPlayerID),
		NewActionConfirmRoundFinished(g.TurnOpponentPlayerID),
	)

	possibleActions := []Action{}
	priority := 0
	for _, action := range allActions {
		action.Enrich(g)
		if !action.IsPossible(g) {
			continue
		}
		if action.GetPriority() < priority {
			continue
		}
		if action.GetPriority() > priority && !action.AllowLowerPriority() {
			priority = action.GetPriority()
			possibleActions = []Action{}
		}
		possibleActions = append(possibleActions, action)
	}
	return possibleActions
}

func SerializeAction(action Action) []byte {
	bs, _ := json.Marshal(action)
	return bs
}

func DeserializeAction(bs []byte) (Action, error) {
	var actionName struct {
		Name string `json:"name"`
	}

	err := json.Unmarshal(bs, &actionName)
	if err != nil {
		return nil, err
	}

	var action Action
	switch actionName.Name {
	case DRAW_FROM_DECK:
		action = &ActionDrawFromDeck{}
	case DRAW_FROM_DISCARD:
		action = &ActionDrawFromDiscard{}
	case DISCARD_CARD:
		action = &ActionDiscardCard{}
	case CLOSE_ROUND:
		action = &ActionClose{}
	case CONFIRM_ROUND_FINISHED:
		action = &ActionConfirmRoundFinished{}
	default:
		return nil, fmt.Errorf("unknown action: [%v]", string(bs))
	}

	err = json.Unmarshal(bs, action)
	if err != nil {
		return nil, err
	}

	return action, nil
}

func _serializeActions(as []Action) []json.RawMessage {
	_as := []json.RawMessage{}
	for _, a := range as {
		_as = append(_as, json.RawMessage(SerializeAction(a)))
	}
	return _as
}

func (g *GameState) ToClientGameState(youPlayerID int) ClientGameState {
	themPlayerID := g.OpponentOf(youPlayerID)

	// GameState may have possible game actions that this player can't take.
	filteredPossibleActions := []Action{}
	for _, a := range g.CalculatePossibleActions() {
		if a.GetPlayerID() == youPlayerID {
			filteredPossibleActions = append(filteredPossibleActions, a)
		}
	}

	var topDiscardCard *Card
	if len(g.DiscardPile) > 0 {
		card := g.DiscardPile[len(g.DiscardPile)-1]
		topDiscardCard = &card
	}

	cgs := ClientGameState{
		RoundNumber:     g.RoundNumber,
		TurnPlayerID:    g.TurnPlayerID,
		YouPlayerID:     youPlayerID,
		ThemPlayerID:    themPlayerID,
		YourScore:       g.Players[youPlayerID].Score,
		TheirScore:      g.Players[themPlayerID].Score,
		YourHand:        g.Players[youPlayerID].Hand.Cards,
		TheirHandSize:   len(g.Players[themPlayerID].Hand.Cards),
		TopDiscardCard:  topDiscardCard,
		DrawPileSize:    g.DrawPile.remainingCards(),
		PossibleActions: _serializeActions(filteredPossibleActions),
		IsGameEnded:     g.IsGameEnded,
		IsRoundFinished: g.IsRoundFinished,
		WinnerPlayerID:  g.WinnerPlayerID,
		LoserPlayerID:   g.LoserPlayerID,
		RuleMaxPoints:   g.RuleMaxPoints,
		HasDrawnCard:    g.HasDrawnCard,
	}

	if len(g.RoundsLog[g.RoundNumber].ActionsLog) > 0 {
		actionsLog := g.RoundsLog[g.RoundNumber].ActionsLog
		cgs.LastActionLog = &actionsLog[len(actionsLog)-1]
	}

	return cgs
}

// ClientGameState represents the state of a Chinchón game as available to a client.
type ClientGameState struct {
	RoundNumber  int `json:"roundNumber"`
	TurnPlayerID int `json:"turnPlayerID"`

	YouPlayerID  int `json:"you"`
	ThemPlayerID int `json:"them"`
	YourScore    int `json:"yourScore"`
	TheirScore   int `json:"theirScore"`

	YourHand       []Card `json:"yourHand"`
	TheirHandSize  int    `json:"theirHandSize"`
	TopDiscardCard *Card  `json:"topDiscardCard"`
	DrawPileSize   int    `json:"drawPileSize"`

	PossibleActions []json.RawMessage `json:"possibleActions"`

	IsGameEnded     bool `json:"isGameEnded"`
	IsRoundFinished bool `json:"isRoundFinished"`

	WinnerPlayerID int `json:"winnerPlayerID"`
	LoserPlayerID  int `json:"loserPlayerID"`

	LastActionLog *ActionLog `json:"lastActionLog"`

	RuleMaxPoints int  `json:"ruleMaxPoints"`
	HasDrawnCard  bool `json:"hasDrawnCard"`
}

type Bot interface {
	ChooseAction(ClientGameState) Action
}
