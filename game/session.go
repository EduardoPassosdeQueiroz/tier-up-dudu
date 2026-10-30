package game

import "crypto/subtle"

// SetSession binds an unguessable credential to the player without disclosing
// it to other participants through the room view.
func (r *Room) SetSession(playerID, token string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p := r.Players[playerID]; p != nil {
		p.SessionToken = token
	}
}

// Resume preserves the seat, secret tier, submitted answer and used powers.
// Disconnected seats remain in the lobby so a suspended browser can return.
func (r *Room) Resume(token string) (string, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if token == "" {
		return "", ""
	}
	for _, id := range r.Order {
		p := r.Players[id]
		if p.SessionToken == "" || subtle.ConstantTimeCompare([]byte(p.SessionToken), []byte(token)) != 1 {
			continue
		}
		p.Connected = true
		hasHost := false
		for _, other := range r.Players {
			if other.Host && other.Connected {
				hasHost = true
				break
			}
		}
		if !hasHost {
			p.Host = true
		}
		return p.ID, p.Name
	}
	return "", ""
}
