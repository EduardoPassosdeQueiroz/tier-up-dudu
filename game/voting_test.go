package game

import (
	"testing"
	"time"
)

func chooseTheme(t *testing.T, room *Room) {
	t.Helper()
	if room.Phase != PhaseVoting || len(room.ThemeOptions) != 5 { t.Fatal("expected five voting options", room.Phase) }
	theme := room.ThemeOptions[0]
	for _, id := range room.Order {
		if player := room.Players[id]; player.Connected && !player.Left {
			if !room.VoteTheme(id, theme, room.Round) { t.Fatal("vote rejected", id) }
		}
	}
	if room.Phase != PhaseAnswering || room.CurrentTheme != theme { t.Fatal("votes did not start answering") }
}

func votingRoom(t *testing.T) *Room {
	t.Helper()
	room := NewRoom("VOTE")
	for _, id := range []string{"h", "a", "b", "c"} { room.AddPlayer(id, id, id == "h") }
	room.StartRound("h")
	for _, id := range room.Order { room.ReadyToPlay(id) }
	return room
}

func TestVotingRequiresEveryoneAndChoosesPlurality(t *testing.T) {
	room := votingRoom(t)
	options := append([]string(nil), room.ThemeOptions...)
	if room.VoteTheme("missing", options[0], 1) || room.VoteTheme("h", "invalid", 1) || room.VoteTheme("h", options[0], 2) { t.Fatal("invalid vote accepted") }
	room.VoteTheme("h", options[0], 1)
	room.VoteTheme("h", options[1], 1)
	room.VoteTheme("a", options[0], 1)
	room.VoteTheme("b", options[0], 1)
	if room.Phase != PhaseVoting || len(room.Votes) != 3 || !room.TimerEnd.IsZero() { t.Fatal("started before final vote") }
	room.VoteTheme("c", options[2], 1)
	if room.CurrentTheme != options[0] || room.Phase != PhaseAnswering { t.Fatal("plurality did not win") }
	if remaining := time.Until(room.TimerEnd); remaining < 119*time.Second || remaining > 120*time.Second { t.Fatal(remaining) }
	if room.VoteTheme("h", options[1], 1) { t.Fatal("late vote accepted") }
}

func TestVotingDisconnectReconnectAndTie(t *testing.T) {
	room := votingRoom(t)
	room.SetSession("c", "token-c")
	room.VoteTheme("c", room.ThemeOptions[4], 1)
	room.Disconnect("c")
	if _, exists := room.Votes["c"]; exists { t.Fatal("disconnected vote counted") }
	room.Resume("token-c")
	room.VoteTheme("h", room.ThemeOptions[0], 1)
	room.VoteTheme("a", room.ThemeOptions[1], 1)
	room.VoteTheme("b", room.ThemeOptions[2], 1)
	if room.Phase != PhaseVoting { t.Fatal("reconnected player was not allowed to vote") }
	room.Disconnect("c")
	valid := false
	for _, theme := range room.ThemeOptions[:3] { if theme == room.CurrentTheme { valid = true } }
	if !valid || room.Phase != PhaseAnswering { t.Fatal("tie winner was not one of the tied options") }
}

func TestFifteenUniqueSuggestionsAndRestartKeepsRoom(t *testing.T) {
	room := votingRoom(t)
	if len(room.MatchThemes) != 15 { t.Fatal("expected exactly fifteen themes") }
	seen := map[string]bool{}
	for round := 1; round <= TotalRounds; round++ {
		for _, theme := range room.ThemeOptions {
			if seen[theme] { t.Fatal("suggestion repeated", theme) }
			seen[theme] = true
		}
		chooseTheme(t, room)
		room.AdvanceExpired(room.TimerEnd)
		for room.Phase == PhaseDiscussing {
			_, judge := room.CurrentDiscussion()
			room.JudgePlace(judge, TierB)
			room.AdvanceExpired(room.TimerEnd)
		}
		if room.Phase != PhaseReview { t.Fatal(room.Phase) }
		room.StartRound("h")
	}
	room.SetSession("a", "token-a")
	if len(seen) != 15 || room.Phase != PhaseFinal || room.Restart("a") { t.Fatal("invalid match end or restart authorization") }
	if !room.Restart("h") || room.Phase != PhaseLobby || room.Code != "VOTE" || len(room.Players) != 4 { t.Fatal("restart lost the room") }
	if len(room.Results) != 0 || len(room.UsedThemes) != 0 || len(room.MatchThemes) != 0 || room.Round != 0 { t.Fatal("restart kept old match data") }
	for _, player := range room.Players { if !player.Connected || player.UsedPower || player.Answered || player.Genius != 0 { t.Fatal("player not reset") } }
	if id, _ := room.Resume("token-a"); id != "a" { t.Fatal("restart lost credential") }
	room.StartRound("h")
	for _, id := range room.Order { room.ReadyToPlay(id) }
	if room.Phase != PhaseVoting || len(room.MatchThemes) != 15 || len(room.UsedThemes) != 5 { t.Fatal("second match did not get a fresh deck") }
	chooseTheme(t, room)
}

func TestUsedPowerAutomaticallyWaits(t *testing.T) {
	room := discussionRoom(t)
	entry, judge := room.CurrentDiscussion()
	active := ""
	for _, id := range room.Order {
		if id != entry.PlayerID { if active == "" { active = id } else { room.Players[id].UsedPower = true } }
	}
	room.JudgePlace(judge, TierB)
	if remaining := time.Until(room.TimerEnd); remaining < 14*time.Second || remaining > 15*time.Second { t.Fatal(remaining) }
	for _, id := range room.Order { if room.Players[id].UsedPower && !room.Players[id].powerChosen { t.Fatal("used power still requires confirmation") } }
	room.SubmitPower(active, 0)
	if room.Phase == PhasePower { t.Fatal("waiting for player who already used a power") }
}

func TestThemeListIsCleanAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, theme := range Themes { if theme == "" || seen[theme] { t.Fatal("invalid or duplicated theme", theme) }; seen[theme] = true }
	for _, theme := range []string{"Comidas de feriados (Natal, Festa Junina, etc.)", "Melhores emojis", "Uma idade ideal que você gostaria de ver seu filho namorando"} { if !seen[theme] { t.Fatal("requested theme missing", theme) } }
}
