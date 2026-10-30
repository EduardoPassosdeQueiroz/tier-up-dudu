package game

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestResumeLobbyPreservesSeat(t *testing.T) {
	r := NewRoom("TEST")
	r.AddPlayer("h", "Host", true)
	r.SetSession("h", "private-token")
	r.Disconnect("h")
	if r.Players["h"] == nil || r.Players["h"].Connected {
		t.Fatal("disconnection removed the seat")
	}
	id, name := r.Resume("private-token")
	if id != "h" || name != "Host" || !r.IsHost("h") || len(r.Order) != 1 {
		t.Fatal("resume did not recover the original host")
	}
	r.Resume("private-token")
	if len(r.Order) != 1 {
		t.Fatal("resume duplicated the player")
	}
	if id, _ := r.Resume("wrong-token"); id != "" {
		t.Fatal("invalid credential was accepted")
	}
	if id, _ := r.Resume(""); id != "" {
		t.Fatal("empty credential was accepted")
	}
	view, _ := json.Marshal(r.ViewForPlayer("h"))
	if strings.Contains(string(view), "private-token") {
		t.Fatal("credential leaked in room view")
	}
}

func TestResumePreservesGameAndMigratedHost(t *testing.T) {
	r := testRoom(t)
	r.SetSession("h", "host-token")
	r.SubmitAnswer("h", "Resposta salva")
	tier, deadline, revision := r.Players["h"].Assigned, r.TimerEnd, r.Revision
	r.Players["h"].UsedPower = true
	r.Disconnect("h")
	if !r.IsHost("a") {
		t.Fatal("host was not transferred")
	}
	id, _ := r.Resume("host-token")
	p := r.Players[id]
	if id != "h" || p.Assigned != tier || p.Answer != "Resposta salva" || !p.Answered || !p.UsedPower || !r.TimerEnd.Equal(deadline) || r.Revision != revision {
		t.Fatal("resuming changed game state")
	}
	if p.Host || !r.IsHost("a") {
		t.Fatal("returning host took ownership away from the new host")
	}
}

func TestKickedSessionCannotResume(t *testing.T) {
	r := NewRoom("TEST")
	r.AddPlayer("h", "Host", true)
	r.AddPlayer("a", "Ana", false)
	r.SetSession("a", "kicked-token")
	if !r.Kick("h", "a") {
		t.Fatal("kick failed")
	}
	if id, _ := r.Resume("kicked-token"); id != "" {
		t.Fatal("removed player resumed")
	}
}

func TestThemeChangeRestartsNinetySeconds(t *testing.T) {
	r := testRoom(t)
	r.ChangeTheme("h")
	if remaining := time.Until(r.TimerEnd); remaining < 89*time.Second || remaining > 90*time.Second {
		t.Fatal("theme change must also use 90 seconds", remaining)
	}
}
