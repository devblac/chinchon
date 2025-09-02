//go:build !tinygo
// +build !tinygo

package exampleclient

import (
	"fmt"

	"github.com/devblac/chinchon/chinchon"
)

func getLastActionString(rs renderState) string {
	if rs.gs.LastActionLog == nil {
		if rs.gs.RoundNumber == 1 {
			return "¡Empezó el juego!"
		}
		return "¡Empezó la ronda!"
	}

	return getActionString(*rs.gs.LastActionLog, rs.gs.YouPlayerID)
}

func getActionString(log chinchon.ActionLog, playerID int) string {
	lastAction, _ := chinchon.DeserializeAction(log.Action)

	who := "Tú"
	if playerID != log.PlayerID {
		who = "Oponente"
	}

	var what string
	switch lastAction.GetName() {
	case chinchon.DRAW_FROM_DECK:
		what = "robó del mazo"
	case chinchon.DRAW_FROM_DISCARD:
		what = "robó de la pila de descarte"
	case chinchon.DISCARD_CARD:
		action := lastAction.(*chinchon.ActionDiscardCard)
		what = fmt.Sprintf("descartó %v", getCardString(action.Card))
	case chinchon.CLOSE_ROUND:
		what = "cerró la ronda"
	case chinchon.CONFIRM_ROUND_FINISHED:
		what = ""
	default:
		what = "???"
	}

	return fmt.Sprintf("%v %v", who, what)
}
