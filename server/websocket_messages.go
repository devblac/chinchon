//go:build !tinygo
// +build !tinygo

package server

import (
	"encoding/json"

	"github.com/devblac/chinchon/chinchon"
)

const (
	MessageTypeHello = iota
	MessageTypeHeresGameState
	MessageTypeAction
	MessageTypeGimmeGameState
)

type IWebsocketMessage[T any] interface {
	GetType() int
	Deserialize() (T, error)
}

type WebsocketMessage struct {
	Type int `json:"type"`
}

func (m WebsocketMessage) GetType() int {
	return m.Type
}

type MessageHello struct {
	WebsocketMessage
	PlayerID int `json:"playerID"`
}

func NewMessageHello(playerID int) MessageHello {
	return MessageHello{WebsocketMessage: WebsocketMessage{Type: MessageTypeHello}, PlayerID: playerID}
}

func (m MessageHello) Deserialize() (int, error) {
	return m.PlayerID, nil
}

type MessageHeresGameState struct {
	WebsocketMessage
	GameState json.RawMessage `json:"gameState"`
}

func NewMessageHeresGameState(gameState chinchon.ClientGameState) (MessageHeresGameState, error) {
	bs, err := json.Marshal(gameState)
	return MessageHeresGameState{WebsocketMessage: WebsocketMessage{Type: MessageTypeHeresGameState}, GameState: bs}, err
}

func (gs MessageHeresGameState) Deserialize() (chinchon.ClientGameState, error) {
	var clientGameState chinchon.ClientGameState
	err := json.Unmarshal(gs.GameState, &clientGameState)
	return clientGameState, err
}

type MessageGimmeGameState struct {
	WebsocketMessage
}

func NewMessageGimmeGameState() MessageGimmeGameState {
	return MessageGimmeGameState{WebsocketMessage: WebsocketMessage{Type: MessageTypeGimmeGameState}}
}

type MessageAction struct {
	WebsocketMessage
	Action json.RawMessage `json:"action"`
}

func NewMessageAction(action chinchon.Action) (MessageAction, error) {
	bs, err := json.Marshal(action)
	return MessageAction{WebsocketMessage: WebsocketMessage{Type: MessageTypeAction}, Action: bs}, err
}

func (a MessageAction) Deserialize() (chinchon.Action, error) {
	return chinchon.DeserializeAction(a.Action)
}
