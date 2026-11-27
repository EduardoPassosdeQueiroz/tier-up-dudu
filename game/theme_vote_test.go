package game

import "testing"

func newThemeVoteRoom(t *testing.T) *Room {
	t.Helper()
	room := NewRoom("VOTE")
	for _, id := range []string{"h", "a", "b"} { room.AddPlayer(id, id, id == "h") }
	room.StartRound("h")
	for _, id := range room.Order { room.Ready(id) }
	if room.Phase != PhaseThemeVote { t.Fatal("expected voting") }
	return room
}

func TestThemeVoteEditRevokesConfirmation(t *testing.T) {
	room := newThemeVoteRoom(t)
	first, second := room.ThemeOptions[0], room.ThemeOptions[1]
	room.SubmitThemeVote("h", first)
	room.SubmitThemeVote("h", second)
	if room.ThemeVotes["h"] != first { t.Fatal("confirmed vote changed without editing") }
	if !room.EditThemeVote("h") || room.ThemeVotes["h"] != "" { t.Fatal("editing did not revoke confirmation") }
	if room.ThemeVoteDrafts["h"] != first { t.Fatal("editing lost previous selection") }
	room.SubmitThemeVote("a", second)
	room.SubmitThemeVote("b", first)
	if room.Phase != PhaseThemeVote { t.Fatal("started while a player was editing") }
	room.SubmitThemeVote("h", second)
	if room.Phase != PhaseAnswering || room.CurrentTheme != second { t.Fatal("reconfirmed vote was not counted") }
	if room.EditThemeVote("h") { t.Fatal("edited after voting ended") }
}

func TestThemeVotesArePrivate(t *testing.T) {
	room := newThemeVoteRoom(t)
	first, second := room.ThemeOptions[0], room.ThemeOptions[1]
	room.SubmitThemeVote("h", first)
	room.SubmitThemeVote("a", second)
	for _, id := range room.Order {
		view := room.ViewForPlayer(id)
		for _, field := range []string{"themeVoteCounts", "themeVotes", "themeVotesNeeded", "ThemeVotes", "ThemeVoteDrafts"} {
			if _, exists := view[field]; exists { t.Fatal("vote data exposed", field) }
		}
		if view["myThemeVote"] != room.ThemeVotes[id] || view["myThemeVoteDraft"] != room.ThemeVoteDrafts[id] { t.Fatal("view contained someone else's choice") }
	}
	room.EditThemeVote("h")
	view := room.ViewForPlayer("h")
	if view["myThemeVote"] != "" || view["myThemeVoteDraft"] != first { t.Fatal("own draft not restored") }
}

func TestThemeVoteEditValidationAndReset(t *testing.T) {
	room := newThemeVoteRoom(t)
	if room.EditThemeVote("missing") || room.EditThemeVote("h") { t.Fatal("invalid edit accepted") }
	room.SubmitThemeVote("h", room.ThemeOptions[0])
	room.Disconnect("h")
	if room.EditThemeVote("h") { t.Fatal("disconnected player edited a vote") }
	room.Round = 2
	room.beginThemeVoteLocked()
	if len(room.ThemeVotes) != 0 || len(room.ThemeVoteDrafts) != 0 { t.Fatal("previous round votes survived") }
	room.Phase = PhaseFinal
	room.ThemeVoteDrafts["a"] = room.ThemeOptions[0]
	if !room.ReturnToLobby("a") || len(room.ThemeVoteDrafts) != 0 { t.Fatal("new match retained draft") }
}
