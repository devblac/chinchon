//go:build !tinygo
// +build !tinygo

package exampleclient

import (
	"fmt"
	"log"
	"strconv"

	"github.com/devblac/chinchon/chinchon"
	"github.com/devblac/chinchon/server"
	"github.com/gorilla/websocket"
)

func Player(playerID int, address string) {
	var (
		ui          = NewUI()
		conn        = handshakeWithServer(playerID, address)
		gameStateCh = recvGameState(conn)

		clientGameState chinchon.ClientGameState
		possibleActions []chinchon.Action
	)
	defer ui.Close()
	defer conn.Close()

	for {
		select {
		case clientGameState = <-gameStateCh:
			if err := ui.render(clientGameState); err != nil {
				log.Fatal(err)
			}
		case key := <-ui.keyCh:
			// If game is over, finish after any key press.
			if clientGameState.IsGameEnded {
				return
			}

			// If there are no possible actions, ignore key presses.
			possibleActions = _deserializeActions(clientGameState.PossibleActions)
			if len(possibleActions) == 0 {
				continue
			}

			// Get the number of the key pressed.
			// If key is not a number, ignore it.
			num, err := strconv.Atoi(string(key))
			if err != nil || num > len(possibleActions) || num <= 0 {
				continue
			}

			// Send the action indicated by the number to the server.
			msg, _ := server.NewMessageAction(possibleActions[num-1])
			if err := server.WsSend(conn, msg); err != nil {
				log.Fatal(err)
			}
		}
	}
}

func handshakeWithServer(playerID int, address string) *websocket.Conn {
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%v/ws", address), nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v", err)
	}

	// Hello message is meant to tell the server who we are, and request game state.
	// Game could be in progress (this could be a reconnection).
	if err := server.WsSend(conn, server.NewMessageHello(playerID)); err != nil {
		log.Fatal(err)
	}

	return conn
}

func recvGameState(conn *websocket.Conn) chan chinchon.ClientGameState {
	gameStateCh := make(chan chinchon.ClientGameState)
	go func() {
		for {
			clientGameState, err := server.WsReadMessage[chinchon.ClientGameState, server.MessageHeresGameState](conn, server.MessageTypeHeresGameState)
			if err != nil {
				log.Fatal(err)
			}
			gameStateCh <- *clientGameState
		}
	}()
	return gameStateCh
}
