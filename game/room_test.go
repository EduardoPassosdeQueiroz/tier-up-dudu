package game

import (
	"encoding/json"
	"testing"
	"time"
)

func testRoom(t *testing.T) *Room {
	t.Helper()
	r := NewRoom("TEST")
	r.AddPlayer("h", "Host", true)
	r.AddPlayer("a", "Ana", false)
	r.AddPlayer("b", "Bia", false)
	if !r.StartRound("h") {
		t.Fatal("start failed")
	}
	for _, id := range r.Order {
		r.Ready(id)
	}
	return r
}
func discussionRoom(t *testing.T) *Room {
	r := testRoom(t)
	for _, id := range r.Order {
		r.SubmitAnswer(id, id+" resposta")
	}
	r.BeginDiscussion()
	return r
}
func TestThemeVotesAndTiming(t *testing.T) {
	r := testRoom(t)
	initial := r.CurrentTheme
	rev := r.Revision
	if left := time.Until(r.TimerEnd); left < 119*time.Second || left > 120*time.Second {
		t.Fatal(left)
	}
	r.SubmitAnswer("h", "rascunho confirmado")
	r.ChangeTheme("a")
	r.ChangeTheme("a")
	if r.CurrentTheme != initial {
		t.Fatal("duplicate vote changed theme")
	}
	r.ChangeTheme("b")
	if r.CurrentTheme == initial || r.Revision != rev+1 || len(r.Answers) != 0 || r.Players["h"].Answered {
		t.Fatal("majority did not reset theme")
	}
	previous := r.CurrentTheme
	r.ChangeTheme("h")
	if r.CurrentTheme != previous {
		t.Fatal("host changed without majority")
	}
	r.ChangeTheme("a")
	if r.CurrentTheme == previous {
		t.Fatal("majority did not change theme")
	}
	r.AdvanceExpired(r.TimerEnd)
	previous = r.CurrentTheme
	r.ChangeTheme("h")
	if r.CurrentTheme != previous {
		t.Fatal("changed outside answering")
	}
	if len(Themes) != 67 {
		t.Fatal(len(Themes))
	}
}
func TestPowersCancelAtEveryTier(t *testing.T) {
	for _, tier := range TierOrder {
		t.Run(string(tier), func(t *testing.T) {
			r := discussionRoom(t)
			entry, judge := r.CurrentDiscussion()
			if !r.JudgePlace(judge, tier) {
				t.Fatal("judge failed")
			}
			if left := time.Until(r.TimerEnd); left < 29*time.Second || left > 30*time.Second {
				t.Fatal(left)
			}
			choice := 1
			for _, id := range r.Order {
				if id != entry.PlayerID {
					r.SubmitPower(id, choice)
					choice = -1
				}
			}
			r.ApplyPowers()
			if entry.Placed != tier || r.Phase != PhaseChanges || len(r.Changes) != 2 {
				t.Fatal("opposite powers must cancel", entry.Placed, r.Phase)
			}
			r.Continue("a")
			if r.Phase != PhaseChanges {
				t.Fatal("non-host continued")
			}
			r.Continue("h")
			if r.Phase != PhaseDiscussing {
				t.Fatal("continue failed")
			}
			board := r.ViewForPlayer("h")["board"].([]any)
			if len(board) != 1 || board[0].(map[string]any)["text"] != entry.Text {
				t.Fatal("board lost placed answer")
			}
		})
	}
}
func TestPowersLimitsAndNoOp(t *testing.T) {
	for _, tc := range []struct {
		tier   Tier
		choice int
		want   Tier
	}{{TierS, 1, TierS}, {TierF, -1, TierF}, {TierE, 1, TierC}, {TierC, -1, TierE}, {TierB, 0, TierB}} {
		r := discussionRoom(t)
		entry, judge := r.CurrentDiscussion()
		r.JudgePlace(judge, tc.tier)
		sent := false
		for _, id := range r.Order {
			if id != entry.PlayerID {
				choice := 0
				if !sent {
					choice = tc.choice
					sent = true
				}
				r.SubmitPower(id, choice)
			}
		}
		r.ApplyPowers()
		if entry.Placed != tc.want {
			t.Fatal(tc, entry.Placed)
		}
		if tc.choice == 0 && r.Phase != PhaseDiscussing {
			t.Fatal("no-op should skip change screen")
		}
		if tc.choice != 0 {
			r.AdvanceExpired(r.TimerEnd)
			if r.Phase != PhaseDiscussing {
				t.Fatal("summary timeout")
			}
		}
	}
}
func TestHostKickAndDisconnect(t *testing.T) {
	r := NewRoom("TEST")
	r.AddPlayer("h", "Host", true)
	r.AddPlayer("a", "Ana", false)
	r.AddPlayer("b", "Bia", false)
	r.AddPlayer("c", "Caio", false)
	if r.Kick("a", "b") || r.Kick("h", "h") {
		t.Fatal("unauthorized kick")
	}
	if !r.Kick("h", "b") {
		t.Fatal("kick failed")
	}
	r.Disconnect("h")
	if !r.IsHost("a") || r.Players["c"].Order != 2 {
		t.Fatal("host order")
	}
	r.AddPlayer("d", "Duda", false)
	if !r.StartRound("a") {
		t.Fatal("new host start")
	}
	if r.Kick("a", "c") || r.StartRound("c") || r.StartRound("a") {
		t.Fatal("phase guards")
	}
	r.Disconnect("a")
	if !r.IsHost("c") {
		t.Fatal("in-game host migration")
	}
}
func TestSecretAndValidation(t *testing.T) {
	r := discussionRoom(t)
	_, judge := r.CurrentDiscussion()
	if r.JudgePlace(judge, Tier("X")) {
		t.Fatal("invalid tier accepted")
	}
	view := r.ViewForPlayer("h")
	blob, _ := json.Marshal(view)
	var obj map[string]any
	json.Unmarshal(blob, &obj)
	if _, ok := obj["results"]; ok {
		t.Fatal("results disclosed early")
	}
	for _, p := range r.Players {
		if TierIndex(p.Assigned) < 0 {
			t.Fatal("invalid assignment")
		}
	}
}
func TestThreeRoundsReachFinal(t *testing.T) {
	r := testRoom(t)
	for round := 1; round <= 3; round++ {
		r.AdvanceExpired(r.TimerEnd)
		for r.Phase == PhaseDiscussing {
			_, judge := r.CurrentDiscussion()
			r.JudgePlace(judge, TierE)
			r.AdvanceExpired(r.TimerEnd)
		}
		if r.Phase != PhaseReview {
			t.Fatal(r.Phase)
		}
		r.StartRound("h")
	}
	if r.Phase != PhaseFinal || len(r.Results) != 3 {
		t.Fatal("final inaccessible")
	}
}

func TestNewRules(t *testing.T) {
	for i, want := range []int{3, 1, -1, -3, -5, -7} {
		if got := ScoreDiff(TierS, TierOrder[i]); got != want {
			t.Fatalf("distance %d: got %d want %d", i, got, want)
		}
	}
	r := NewRoom("TEST")
	r.AddPlayer("h", "Host", true)
	r.AddPlayer("a", "Ana", false)
	r.AddPlayer("b", "Bia", false)
	r.StartRound("h")
	r.Ready("h")
	r.Ready("a")
	if r.Phase != PhaseInstructions {
		t.Fatal("started before everyone ready")
	}
	r.Ready("b")
	r.SubmitAnswer("h", "primeira")
	r.EditAnswer("h")
	if r.Players["h"].Answered || len(r.Answers) != 0 {
		t.Fatal("edit did not revoke confirmation")
	}
	r.SubmitAnswer("h", "nova")
	if len(r.Answers) != 1 || r.Answers[0].Text != "nova" {
		t.Fatal("edited answer not saved")
	}
	r.Phase = PhaseReview
	r.Results = []*RoundResult{{Answers: r.Answers}}
	r.Like("h", 0, "h")
	if len(r.Answers[0].Likes) != 0 {
		t.Fatal("self like")
	}
	r.Like("a", 0, "h")
	if !r.Answers[0].Likes["a"] {
		t.Fatal("like missing")
	}
	r.Like("a", 0, "h")
	if len(r.Answers[0].Likes) != 0 {
		t.Fatal("unlike failed")
	}
}

func TestGeniusDoesNotChangeWhenOriginalAlreadyCorrect(t *testing.T) {
	r := discussionRoom(t)
	entry, judge := r.CurrentDiscussion()
	entry.Original = TierS
	r.JudgePlace(judge, TierS)
	var actor string
	for _, id := range r.Order {
		if id != entry.PlayerID {
			if actor == "" {
				actor = id
				r.SubmitPower(id, 1)
			} else {
				r.SubmitPower(id, 0)
			}
		}
	}
	if r.Players[actor].Genius != 0 {
		t.Fatal("power on an already correct answer must not change individual score")
	}
}
func TestThemeDrawCanReuseEarlierThemes(t *testing.T) {
	r := NewRoom("TEST")
	r.CurrentTheme = Themes[0]
	r.UsedThemes = append([]string(nil), Themes...)
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		theme := r.pickTheme()
		if theme == r.CurrentTheme {
			t.Fatal("theme change repeated current theme")
		}
		seen[theme] = true
	}
	if len(seen) < 2 {
		t.Fatal("draw did not select among earlier themes")
	}
}

func TestDuplicatePlayerNames(t *testing.T) {
	r := NewRoom("TEST")
	r.AddPlayer("a", "Enzo", true)
	if _, reason := r.AddPlayerChecked("b", " enZO ", false); reason != "nome_duplicado" {
		t.Fatalf("duplicate name accepted: %q", reason)
	}
	r.Disconnect("a")
	if _, reason := r.AddPlayerChecked("c", "ENZO", false); reason != "nome_duplicado" {
		t.Fatal("disconnected seat must retain its name")
	}
}

func TestGeniusPenalizesMovingAwayFromCorrectTier(t *testing.T) {
	r := discussionRoom(t)
	entry, judge := r.CurrentDiscussion()
	entry.Original = TierB
	r.JudgePlace(judge, TierB)
	actor := ""
	for _, id := range r.Order {
		if id != entry.PlayerID {
			if actor == "" {
				actor = id
				r.SubmitPower(id, 1)
			} else {
				r.SubmitPower(id, 0)
			}
		}
	}
	if got := r.Players[actor].Genius; got != -2 {
		t.Fatalf("got %d, want -2", got)
	}
}
