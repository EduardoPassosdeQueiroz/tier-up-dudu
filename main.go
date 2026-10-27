package main

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"tierlist/game"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	rooms   = make(map[string]*game.Room)
	roomsMu sync.RWMutex

	conns    = make(map[string]*websocket.Conn) // playerID -> conn
	connRoom = make(map[string]*game.Room)      // playerID -> room
	connsMu  sync.RWMutex

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			host := r.Host
			return strings.HasSuffix(origin, "://"+host)
		},
	}
)

type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type JoinMsg struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type AnswerMsg struct {
	Text string `json:"text"`
}

type JudgeMsg struct {
	Tier string `json:"tier"`
}

type PowerMsg struct {
	Choice int `json:"choice"`
}

func generateCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 4)
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "TIER"
	}
	for i := range code {
		code[i] = chars[int(b[i])%len(chars)]
	}
	return string(code)
}

func createRoom() *game.Room {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	code := generateCode()
	for {
		if _, exists := rooms[code]; !exists {
			room := game.NewRoom(code)
			rooms[code] = room
			return room
		}
		code = generateCode()
	}
}

func getRoom(code string) *game.Room {
	roomsMu.RLock()
	defer roomsMu.RUnlock()
	return rooms[code]
}

func broadcastView(room *game.Room) {
	connsMu.RLock()
	defer connsMu.RUnlock()
	for _, p := range room.Players {
		conn := conns[p.ID]
		if conn == nil {
			continue
		}
		view := room.ViewForPlayer(p.ID)
		if view == nil {
			continue
		}
		if err := conn.WriteJSON(view); err != nil {
			log.Printf("write error para %s: %v", p.ID, err)
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	var room *game.Room
	var playerID string

	defer func() {
		conn.Close()
		if playerID != "" {
			connsMu.Lock()
			delete(conns, playerID)
			delete(connRoom, playerID)
			connsMu.Unlock()
			if room != nil {
				broadcastView(room)
			}
		}
	}()

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg WSMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "join":
			var join JoinMsg
			json.Unmarshal(msg.Data, &join)
			join.Name = strings.TrimSpace(join.Name)
			join.Code = strings.ToUpper(strings.TrimSpace(join.Code))
			if join.Name == "" || len(join.Name) > 20 {
				conn.WriteJSON(map[string]string{"error": "nome_invalido"})
				continue
			}

			if join.Code == "" {
				room = createRoom()
				playerID = uuid.New().String()
				room.AddPlayer(playerID, join.Name, true)
			} else {
				room = getRoom(join.Code)
				if room == nil {
					conn.WriteJSON(map[string]string{"error": "sala_nao_encontrada"})
					continue
				}
				playerID = uuid.New().String()
				room.AddPlayer(playerID, join.Name, false)
			}

			connsMu.Lock()
			conns[playerID] = conn
			connRoom[playerID] = room
			connsMu.Unlock()

			conn.WriteJSON(map[string]any{
				"type":  "joined",
				"code":  room.Code,
				"host":  room.Players[playerID].Host,
				"phase": string(room.Phase),
			})
			broadcastView(room)

		case "start":
			if room == nil {
				continue
			}
			room.StartRound()
			broadcastView(room)

		case "answer":
			if room == nil {
				continue
			}
			var ans AnswerMsg
			json.Unmarshal(msg.Data, &ans)
			allAnswered := room.SubmitAnswer(playerID, ans.Text)
			if allAnswered {
				room.BeginDiscussion()
			}
			broadcastView(room)

		case "judge":
			if room == nil {
				continue
			}
			var j JudgeMsg
			json.Unmarshal(msg.Data, &j)
			room.JudgePlace(playerID, game.Tier(j.Tier))
			broadcastView(room)

		case "power":
			if room == nil {
				continue
			}
			var p PowerMsg
			json.Unmarshal(msg.Data, &p)
			allPowers := room.SubmitPower(playerID, p.Choice)
			if allPowers {
				room.ApplyPowers()
			}
			broadcastView(room)

		case "nextRound":
			if room == nil {
				continue
			}
			room.StartRound()
			broadcastView(room)

		case "changeTheme":
			if room != nil {
				room.ChangeTheme(playerID)
				broadcastView(room)
			}
		}
	}
}

func main() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", fs)
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	go runExpiryTicker()
	log.Printf("Tier Up rodando em http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func runExpiryTicker() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for now := range ticker.C {
		roomsMu.RLock()
		allRooms := make([]*game.Room, 0, len(rooms))
		for _, room := range rooms {
			allRooms = append(allRooms, room)
		}
		roomsMu.RUnlock()
		for _, room := range allRooms {
			if room.AdvanceExpired(now) {
				broadcastView(room)
			}
		}
	}
}
