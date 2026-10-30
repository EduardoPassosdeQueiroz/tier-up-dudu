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
 writeMu sync.Mutex
 broadcastMu sync.Mutex

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
	Token string `json:"token"`
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

func writeJSON(conn *websocket.Conn, value any) error {
 writeMu.Lock(); defer writeMu.Unlock(); conn.SetWriteDeadline(time.Now().Add(5*time.Second)); return conn.WriteJSON(value)
}

func broadcastView(room *game.Room) {
 broadcastMu.Lock()
 defer broadcastMu.Unlock()
	connsMu.RLock()
	defer connsMu.RUnlock()
	for _, id := range room.PlayerIDs() {
		conn := conns[id]
		if conn == nil {
			continue
		}
		view := room.ViewForPlayer(id)
		if view == nil {
			continue
		}
		if err := writeJSON(conn, view); err != nil {
			log.Printf("write error para %s: %v", id, err)
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	conn.SetReadLimit(4096)
 conn.SetReadDeadline(time.Now().Add(60*time.Second))
 conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(60*time.Second)) })
 done:=make(chan struct{})
 defer close(done)
 go func() {
 ticker:=time.NewTicker(20*time.Second); defer ticker.Stop()
 for { select { case <-done: return; case <-ticker.C:
 if err:=conn.WriteControl(websocket.PingMessage,nil,time.Now().Add(5*time.Second)); err!=nil {conn.Close();return}
 } }
 }()
 var room *game.Room
	var playerID string

	defer func() {
		conn.Close()
		if playerID != "" {
			connsMu.Lock()
			// An old socket must not disconnect a newly resumed session.
			current := conns[playerID] == conn
			if current {
				delete(conns, playerID)
				delete(connRoom, playerID)
				if room != nil {
					room.Disconnect(playerID)
				}
			}
			connsMu.Unlock()
			if current && room != nil {
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
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if playerID != "" {
			connsMu.RLock()
			current := conns[playerID] == conn
			connsMu.RUnlock()
			if !current { return }
		}

		switch msg.Type {
		case "join":
 if playerID != "" { continue }
			var join JoinMsg
			json.Unmarshal(msg.Data, &join)
			join.Name = strings.TrimSpace(join.Name)
			join.Code = strings.ToUpper(strings.TrimSpace(join.Code))
			if join.Token == "" && (join.Name == "" || len([]rune(join.Name)) > 20) {
				writeJSON(conn, map[string]string{"error": "nome_invalido"})
				continue
			}

			var previous *websocket.Conn
			if join.Token != "" {
				room = getRoom(join.Code)
				if room == nil {
					writeJSON(conn, map[string]string{"error": "sessao_expirada"})
					return
				}
				connsMu.Lock()
				playerID, join.Name = room.Resume(join.Token)
				if playerID != "" {
					previous = conns[playerID]
					conns[playerID] = conn
					connRoom[playerID] = room
				}
				connsMu.Unlock()
				if playerID == "" {
					writeJSON(conn, map[string]string{"error": "sessao_expirada"})
					return
				}
			} else if join.Code == "" {
				room = createRoom()
				playerID = uuid.New().String()
				room.AddPlayer(playerID, join.Name, true)
			} else {
				room = getRoom(join.Code)
				if room == nil {
					writeJSON(conn, map[string]string{"error": "sala_nao_encontrada"})
					continue
				}
				playerID = uuid.New().String()
				if room.AddPlayer(playerID, join.Name, false)==nil { writeJSON(conn,map[string]string{"error":"partida_em_andamento"}); playerID=""; room=nil; continue }
			}
			if join.Token == "" {
				join.Token = uuid.New().String()
				room.SetSession(playerID, join.Token)
			}

			connsMu.Lock()
			conns[playerID] = conn
			connRoom[playerID] = room
			connsMu.Unlock()
			if previous != nil && previous != conn {
				writeJSON(previous, map[string]string{"type": "sessionReplaced"})
				previous.Close()
			}

			writeJSON(conn, map[string]any{
				"type":  "joined",
				"code":  room.Code,
				"host":  room.IsHost(playerID),
				"phase": room.ViewForPlayer(playerID)["phase"],
				"token": join.Token,
				"name": join.Name,
			})
			broadcastView(room)

		case "ping":
			writeJSON(conn, map[string]string{"type": "pong"})
		case "start":
			if room == nil {
				continue
			}
			room.StartRound(playerID)
			broadcastView(room)

		case "answer":
			if room == nil {
				continue
			}
			var ans AnswerMsg
			json.Unmarshal(msg.Data, &ans)
			room.SubmitAnswer(playerID, ans.Text)
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
			room.SubmitPower(playerID, p.Choice)
			broadcastView(room)

		case "nextRound":
			if room == nil {
				continue
			}
			room.StartRound(playerID)
			broadcastView(room)

		case "continue":
 if room != nil { room.Continue(playerID); broadcastView(room) }
 case "kick":
 var target struct { PlayerID string `json:"playerID"` }; json.Unmarshal(msg.Data,&target)
 if room!=nil && room.Kick(playerID,target.PlayerID) {
 connsMu.RLock(); kicked:=conns[target.PlayerID]; connsMu.RUnlock()
 if kicked!=nil { writeJSON(kicked,map[string]string{"error":"removido"}); kicked.Close() }
 broadcastView(room)
 }
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
	http.Handle("/static/", http.StripPrefix("/static/", fs))
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
	log.Printf("Tier Du rodando em http://localhost:%s", port)
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
