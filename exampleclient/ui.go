//go:build !tinygo
// +build !tinygo

package exampleclient

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/devblac/chinchon/chinchon"
	"github.com/nsf/termbox-go"
)

type ui struct {
	keyCh chan rune
}

func NewUI() *ui {
	ui := &ui{}
	ui.keyCh = ui.startKeyEventLoop()
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	return ui
}

func (u *ui) Close() {
	termbox.Close()
}

type renderMode int

const (
	PRINT_MODE_NORMAL renderMode = iota
	PRINT_MODE_SHOW_ROUND_RESULT
	PRINT_MODE_END
)

type renderState struct {
	mode            renderMode
	viewportWidth   int
	viewportHeight  int
	gs              chinchon.ClientGameState
	possibleActions []chinchon.Action
}

func calculateRenderState(state chinchon.ClientGameState) renderState {
	var (
		viewportWidth, viewportHeight = termbox.Size()
		possibleActions               = _deserializeActions(state.PossibleActions)
		gs                            = state
		mode                          = PRINT_MODE_NORMAL
	)
	if state.IsRoundFinished {
		mode = PRINT_MODE_SHOW_ROUND_RESULT
	}
	if state.IsGameEnded {
		mode = PRINT_MODE_END
	}

	return renderState{
		mode:            mode,
		gs:              gs,
		possibleActions: possibleActions,
		viewportWidth:   viewportWidth,
		viewportHeight:  viewportHeight,
	}
}

func (u *ui) render(state chinchon.ClientGameState) error {
	if err := termbox.Clear(termbox.ColorWhite, termbox.ColorBlack); err != nil {
		return err
	}

	rs := calculateRenderState(state)

	renderScores(rs)
	renderTheirHand(rs)
	renderDiscardPile(rs)
	renderDrawPile(rs)
	renderLastAction(rs)
	renderEndSummary(rs)
	renderYourHand(rs)
	renderActions(rs)

	termbox.Flush()
	// This is an artificial delay to make the game more human-like.
	time.Sleep(1 * time.Second)

	return nil
}

func renderScores(rs renderState) {
	renderUpToAt(rs.viewportWidth-1, 0, fmt.Sprintf("Ronda número %d", rs.gs.RoundNumber))

	renderUpToAt(rs.viewportWidth-1, 1, fmt.Sprintf("Tus puntos: %d", rs.gs.YourScore))
	renderUpToAt(rs.viewportWidth-1, 2, fmt.Sprintf("Sus puntos: %d", rs.gs.TheirScore))
}

func renderTheirHand(rs renderState) {
	displayText := fmt.Sprintf("Cartas del oponente: %d cartas", rs.gs.TheirHandSize)
	renderAt(0, 4, displayText)
}

func renderDiscardPile(rs renderState) {
	displayText := "Pila de descarte: "
	if rs.gs.TopDiscardCard != nil {
		displayText += getCardString(*rs.gs.TopDiscardCard)
	} else {
		displayText += "(vacía)"
	}
	renderAt(0, rs.viewportHeight/2-2, displayText)
}

func renderDrawPile(rs renderState) {
	displayText := fmt.Sprintf("Pila de robo: %d cartas", rs.gs.DrawPileSize)
	renderAt(0, rs.viewportHeight/2-1, displayText)
}

func renderLastAction(rs renderState) {
	renderAt(0, rs.viewportHeight/2, getLastActionString(rs))
}

func renderEndSummary(rs renderState) {
	var renderText string

	switch rs.mode {
	case PRINT_MODE_SHOW_ROUND_RESULT:
		renderText = "Ronda terminada. Presiona cualquier tecla para continuar."
	case PRINT_MODE_END:
		var resultText string
		if rs.gs.YouPlayerID == rs.gs.WinnerPlayerID {
			resultText = "¡Ganaste! 🥰"
		} else {
			resultText = "Perdiste 😭"
		}
		renderText = fmt.Sprintf("%v", resultText)
	}

	renderAt(0, rs.viewportHeight/2, renderText)
}

func renderYourHand(rs renderState) {
	displayText := "Tus cartas: " + getCardsString(rs.gs.YourHand)
	renderAt(0, rs.viewportHeight-4, displayText)
}

func renderActions(rs renderState) {
	var renderText string

	actionsString := ""
	for i, action := range rs.possibleActions {
		actionsString += fmt.Sprintf("%d. %s   ", i+1, action.String())
	}
	renderText = actionsString

	if len(rs.possibleActions) == 0 {
		renderText = "Esperando al otro jugador..."
	}

	if rs.mode == PRINT_MODE_END {
		renderText = "Presiona cualquier tecla para continuar..."
	}

	renderAt(0, rs.viewportHeight-2, renderText)
}

func renderAt(x, y int, s string) {
	_s := []rune(s)
	for i, r := range _s {
		termbox.SetCell(x+i, y, r, termbox.ColorDefault, termbox.ColorDefault)
	}
}

// Write so that the output ends at x, y
func renderUpToAt(x, y int, s string) {
	_s := []rune(s)
	for i, r := range _s {
		termbox.SetCell(x-len(_s)+i, y, r, termbox.ColorDefault, termbox.ColorDefault)
	}
}

func getCardsString(cards []chinchon.Card) string {
	var cs []string
	for _, card := range cards {
		cs = append(cs, getCardString(card))
	}
	return strings.Join(cs, "  ")
}

func getCardString(card chinchon.Card) string {
	return fmt.Sprintf("[%v%v]", card.Number, suitEmoji(card.Suit))
}

func suitEmoji(suit string) string {
	switch suit {
	case chinchon.ESPADA:
		return "🗡️"
	case chinchon.BASTO:
		return "🌿"
	case chinchon.ORO:
		return "💰"
	case chinchon.COPA:
		return "🍷"
	default:
		return "❓"
	}
}

func _deserializeActions(as []json.RawMessage) []chinchon.Action {
	_as := []chinchon.Action{}
	for _, a := range as {
		_a, _ := chinchon.DeserializeAction(a)
		_as = append(_as, _a)
	}
	return _as
}
