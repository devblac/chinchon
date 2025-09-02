package newbot

import (
	"encoding/json"

	"github.com/devblac/chinchon/chinchon"
)

func _deserializeActions(as []json.RawMessage) []chinchon.Action {
	_as := []chinchon.Action{}
	for _, a := range as {
		_a, _ := chinchon.DeserializeAction(a)
		_as = append(_as, _a)
	}
	return _as
}

// Utility functions for the simplified Chinchón bot
