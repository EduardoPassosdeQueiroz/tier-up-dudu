package game

import (
	"math/rand"
	"time"
)

func (r *Room) prepareThemesLocked() {
	deck := append([]string(nil), Themes...)
	rand.Shuffle(len(deck), func(first, second int) { deck[first], deck[second] = deck[second], deck[first] })
	r.MatchThemes = append([]string(nil), deck[:TotalRounds*5]...)
	r.UsedThemes = nil
}

func (r *Room) VoteTheme(playerID, theme string, round int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	player := r.Players[playerID]
	if player == nil || player.Left || !player.Connected || r.Phase != PhaseVoting || round != r.Round { return false }
	valid := false
	for _, option := range r.ThemeOptions { if option == theme { valid = true; break } }
	if !valid { return false }
	r.Votes[playerID] = theme
	r.finishVotingLocked()
	return true
}

func (r *Room) finishVotingLocked() {
	if r.Phase != PhaseVoting || r.connectedCountLocked() == 0 { return }
	for id, player := range r.Players {
		if player.Connected && !player.Left && r.Votes[id] == "" { return }
	}
	counts := map[string]int{}
	for id, theme := range r.Votes {
		if player := r.Players[id]; player != nil && player.Connected && !player.Left { counts[theme]++ }
	}
	best := -1
	winners := []string{}
	for _, theme := range r.ThemeOptions {
		if counts[theme] > best { best = counts[theme]; winners = []string{theme} } else if counts[theme] == best { winners = append(winners, theme) }
	}
	r.CurrentTheme = winners[rand.Intn(len(winners))]
	r.assignTiers()
	r.Phase = PhaseAnswering
	r.TimerEnd = time.Now().Add(AnswerDuration)
}

func (r *Room) Restart(playerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	player := r.Players[playerID]
	if player == nil || player.Left || !player.Connected || !player.Host || r.Phase != PhaseFinal { return false }
	for _, id := range append([]string(nil), r.Order...) {
		if r.Players[id].Left { r.removeLobbyPlayerLocked(id); continue }
		member := r.Players[id]
		member.Assigned = ""
		member.Answer = ""
		member.Answered = false
		member.UsedPower = false
		member.PowerChoice = 0
		member.powerChosen = false
		member.Ready = false
		member.Genius = 0
	}
	r.Phase = PhaseLobby
	r.Round = 0
	r.Revision++
	r.CurrentTheme = ""
	r.MatchThemes = nil
	r.ThemeOptions = nil
	r.UsedThemes = nil
	r.Votes = nil
	r.ReadyState = nil
	r.IntroPlayers = nil
	r.Answers = nil
	r.Discussion = nil
	r.DiscIdx = 0
	r.Results = nil
	r.Changes = nil
	r.TimerEnd = time.Time{}
	return true
}
