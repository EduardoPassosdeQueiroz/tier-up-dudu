package game

type PowerAction struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Choice     int    `json:"choice"`
	Points     int    `json:"points"`
}

func (r *Room) ReadyToPlay(playerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[playerID]
	if p == nil || p.Left || !p.Connected || r.Phase != PhaseInstructions {
		return false
	}
	r.ReadyState[playerID] = true
	r.IntroPlayers[playerID] = true
	return r.beginWhenReadyLocked()
}

func (r *Room) beginWhenReadyLocked() bool {
	if r.connectedCountLocked() == 0 { return false }
	for id := range r.IntroPlayers {
		if p:=r.Players[id]; p!=nil && !p.Left && !r.ReadyState[id] { return false }
	}
	r.startRoundLocked()
	return true
}

// Reopening an answer revokes confirmation without extending the shared timer.
func (r *Room) EditAnswer(playerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[playerID]
	if p == nil || p.Left || !p.Connected || r.Phase != PhaseAnswering || !p.Answered { return false }
	p.Answered = false
	for i, a := range r.Answers {
		if a.PlayerID == playerID { r.Answers = append(r.Answers[:i], r.Answers[i+1:]...); break }
	}
	return true
}

func unansweredText(text string) string {
	if text != "" { return text }
	return "(sem resposta)"
}

// Explicit departure revokes the credential. Temporary disconnection does not.
func (r *Room) Leave(playerID string) {
	r.Disconnect(playerID)
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[playerID]
	if p == nil { return }
	p.Left = true
	p.SessionToken = ""
	delete(r.ReadyState, playerID)
	delete(r.IntroPlayers, playerID)
	if r.Phase == PhaseInstructions { r.beginWhenReadyLocked() }
	if r.Phase == PhaseLobby { r.removeLobbyPlayerLocked(playerID) }
}

func (r *Room) ToggleLike(playerID string, round int, answerPlayerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[playerID]
	if p == nil || p.Left || !p.Connected || (r.Phase != PhaseReview && r.Phase != PhaseFinal) || round < 1 || round > len(r.Results) { return false }
	for _, a := range r.Results[round-1].Answers {
		if a.PlayerID != answerPlayerID { continue }
		if a.PlayerID == playerID { return false }
		if a.Likes == nil { a.Likes = map[string]bool{} }
		if a.Likes[playerID] { delete(a.Likes, playerID) } else { a.Likes[playerID] = true }
		return true
	}
	return false
}

func (r *Room) resultViewLocked(result *RoundResult, round int, viewerID string) map[string]any {
	answers := []map[string]any{}
	for _, a := range result.Answers {
		answers = append(answers, map[string]any{
			"playerId": a.PlayerID, "playerName": a.PlayerName, "text": a.Text,
			"original": a.Original, "placed": a.Placed, "leaderName": a.LeaderName,
			"score": ScoreDiff(a.Original, a.Placed), "likeCount": len(a.Likes),
			"myLiked": a.Likes[viewerID], "canLike": viewerID != a.PlayerID,
		})
	}
	return map[string]any{"round": round, "theme": result.Theme, "score": result.Score, "answers": answers}
}

func (r *Room) statsLocked() []map[string]any {
	// Names come from historical answers as well: departed players keep their scores.
	names := map[string]string{}
	scores, powers, uses := map[string]int{}, map[string]int{}, map[string]int{}
	for _, result := range r.Results {
		for _, a := range result.Answers {
			names[a.PlayerID] = a.PlayerName
			scores[a.PlayerID] += ScoreDiff(a.Original, a.Placed)
			for _, action := range a.Powers {
				names[action.PlayerID] = action.PlayerName
				powers[action.PlayerID] += action.Points
				uses[action.PlayerID]++
			}
		}
	}
	stats := []map[string]any{}
	for _, id := range r.Order {
		if name, ok := names[id]; ok { stats = append(stats, map[string]any{"playerId": id, "playerName": name, "score": scores[id], "powerScore": powers[id], "powerUses": uses[id]}) }
	}
	return stats
}
