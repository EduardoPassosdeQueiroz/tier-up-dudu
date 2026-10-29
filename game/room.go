package game

import (
	"math/rand"
 "encoding/json"
 "strings"
	"sync"
	"time"
)

type Tier string

const (
	TierS Tier = "S"
	TierA Tier = "A"
	TierB Tier = "B"
	TierC Tier = "C"
	TierE Tier = "E"
 TierF Tier = "F"
)

var TierOrder = []Tier{TierS, TierA, TierB, TierC, TierE, TierF}

func TierIndex(t Tier) int {
	for i, v := range TierOrder {
		if v == t {
			return i
		}
	}
	return -1
}

func ShiftTier(t Tier, delta int) Tier {
	i := TierIndex(t) + delta
	if i < 0 {
		i = 0
	}
	if i > len(TierOrder)-1 {
		i = len(TierOrder) - 1
	}
	return TierOrder[i]
}

type Phase string

const (
	PhaseLobby      Phase = "lobby"
	PhaseAnswering  Phase = "answering"
	PhaseDiscussing Phase = "discussing"
	PhasePower      Phase = "power"
	PhaseChanges Phase = "changes"
 PhaseReview     Phase = "review"
	PhaseFinal      Phase = "final"
)

type Player struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Host      bool   `json:"host"`
	Order     int    `json:"order"`
	Connected bool   `json:"connected"`

	Assigned    Tier   `json:"-"`
	Answer      string `json:"-"`
	Answered    bool   `json:"answered"`
	UsedPower   bool   `json:"usedPower"`
	PowerChoice int    `json:"-"`
	powerChosen bool   `json:"-"`
}

type AnswerEntry struct {
 LeaderName string `json:"leaderName,omitempty"`
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Text       string `json:"text"`
	Original   Tier   `json:"original,omitempty"`
	Placed     Tier   `json:"placed,omitempty"`
	PlacedSet  bool   `json:"placedSet"`
}

type RoundResult struct {
	Theme   string         `json:"theme"`
	Score   int            `json:"score"`
	Answers []*AnswerEntry `json:"answers"`
}

type Room struct {
	mu      sync.RWMutex
	Code    string
	Players map[string]*Player
	Order   []string
	Phase   Phase

	Round        int
	UsedThemes   []string
	CurrentTheme string

	Answers    []*AnswerEntry
	Discussion []int
	DiscIdx    int

	TimerEnd time.Time

	ThemeVotes map[string]bool
 Revision int
 Changes []map[string]any
 Results []*RoundResult
}

func NewRoom(code string) *Room {
	return &Room{
		Code:    code,
		Players: map[string]*Player{},
		Phase:   PhaseLobby,
	}
}

func (r *Room) AddPlayer(id, name string, host bool) *Player {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Phase != PhaseLobby { return nil }
 p := &Player{
		ID:        id,
		Name:      name,
		Host:      host || len(r.Order)==0,
		Order:     len(r.Order),
		Connected: true,
	}
	r.Players[id] = p
	r.Order = append(r.Order, id)
	return p
}

func (r *Room) PlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Players)
}

func (r *Room) IsHost(playerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p := r.Players[playerID]
	return p != nil && p.Host
}

func (r *Room) judgeForAnswer(ownerID string) string {
	owner := r.Players[ownerID]
	if owner == nil || len(r.Order) == 0 {
		return ""
	}
	for offset := 1; offset <= len(r.Order); offset++ {
 id := r.Order[(owner.Order + offset) % len(r.Order)]
 if p := r.Players[id]; p != nil && p.Connected { return id }
 }
 return ""
}

func (r *Room) assignTiers() {
	tiers := append([]Tier(nil), TierOrder...)
 rand.Shuffle(len(tiers), func(i,j int) { tiers[i],tiers[j]=tiers[j],tiers[i] })
 i := 0
	for _, p := range r.Players {
		p.Assigned = tiers[i % len(tiers)]
 i++
		p.Answer = ""
		p.Answered = false
		p.UsedPower = false
		p.PowerChoice = 0
		p.powerChosen = false
	}
}

var Themes = []string{
"Comidas típicas de Natal",
"Frutas tropicais",
"Doces/Salgados de aniversário",
"Comidas de festa junina",
"Sabores de pizza",
"Bebidas",
"Sobremesas com chocolate",
"Animais marinhos",
"Animais de fazenda",
"Animais extintos",
"Insetos",
"Vilões da Disney",
"Personagens da Marvel/DC",
"Youtuberes",
"Personagens que dão mais medo em filmes",
"Séries",
"Personagens com mais aura",
"Personagens de séries de TV Aberta",
"Filmes de animação",
"Lugares para se esconder num esconde-esconde mortal",
"Lugares para passar a lua de mel",
"Lugares para levar alguém em um primeiro encontro horrível",
"Melhores pontos turísticos",
"Lugar para você dormir",
"Personagens de Clash Royale",
"Melhores Mobs de Minecraft",
"Jogos de tabuleiro",
"Jogos On-line",
"Melhores personagens fictícios",
"Coisas que você levaria para sobreviver em uma ilha deserta",
"Coisas que seriam suspeitas de encontrar no porta-malas de um carro",
"Objetos que seriam valiosos após o fim do mundo",
"Desculpas para chegar atrasado",
"Coisas que não se deve falar em um casamento",
"Motivos para terminar um namoro",
"Objetos que podem machucar se caírem na sua cabeça",
"Presentes ruins de aniversário",
"Coisas que você não deveria pesquisar no Google",
"Jogadores brasileiros de futebol",
"Maiores times do Brasil",
"Melhores esportes olímpicos de se jogar",
"Celebridade que você não deixaria cuidar do seu filho",
"Você ganharia desse cara numa luta?",
"Bandas e Cantores",
"Super-poderes",
"Dinossauros",
"Invenções que não deveriam existir",
"Invenções que mudaram o mundo",
"Órgãos do corpo humano",
"Objetos atuais que assustariam pessoas da Idade Média",
"O que você faria se pudesse parar o tempo",
"Coisas que você faria se pudesse viajar no tempo",
"Objetos de cozinha mais úteis",
"Materiais escolares",
"É trair (S) ou não (F)",
"Tratamento de princesa (S) ou mínimo",
"Conhecidos bonitos (quem namora não discute)",
"Red Flag (S) e Green Flag (F)",
"Motivo de orgulho (S) ou vergonha (F)",
"Melhores profissões (desconsidere salário)",
"Coisas que custam mais de R$ 1 milhão",
"Coisas que seriam úteis em um apocalipse zumbi",
"Coisas que podem causar o fim da humanidade",
"Personagens de animação mais atraentes",
}

func (r *Room) pickTheme() string {
	used := map[string]bool{}
	for _, t := range r.UsedThemes {
		used[t] = true
	}
	avail := []string{}
	for _, t := range Themes {
		if !used[t] {
			avail = append(avail, t)
		}
	}
	if len(avail) == 0 {
		r.UsedThemes = []string{r.CurrentTheme}
 for _,t := range Themes { if t != r.CurrentTheme { avail=append(avail,t) } }
 return avail[rand.Intn(len(avail))]
	}
	return avail[rand.Intn(len(avail))]
}

func (r *Room) StartRound(requesterID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p := r.Players[requesterID]; p == nil || !p.Host || (r.Phase != PhaseLobby && r.Phase != PhaseReview) { return false }
 if r.Phase == PhaseLobby && r.connectedCountLocked() < 3 { return false }
 if r.Round >= 5 {
		r.Phase = PhaseFinal
		return false
	}
	r.Round++
 r.Revision++
 r.ThemeVotes = map[string]bool{}
 r.Changes = nil
	theme := r.pickTheme()
	r.UsedThemes = append(r.UsedThemes, theme)
	r.CurrentTheme = theme
	r.Answers = nil
	r.Discussion = nil
	r.DiscIdx = 0
	r.assignTiers()
	r.Phase = PhaseAnswering
	r.TimerEnd = time.Now().Add(80 * time.Second)
	return true
}

func (r *Room) SubmitAnswer(playerID, text string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[playerID]
	if p == nil || !p.Connected || p.Answered || r.Phase != PhaseAnswering {
		return false
	}
	text = strings.TrimSpace(text)
 if text == "" || len([]rune(text)) > 100 { return false }
 p.Answer = text
	p.Answered = true
	r.Answers = append(r.Answers, &AnswerEntry{
		PlayerID:   p.ID,
		PlayerName: p.Name,
		Text:       text,
		Original:   p.Assigned,
	})
	if r.allAnsweredLocked() { r.beginDiscussionLocked(); return true }; return false
}

func (r *Room) allAnsweredLocked() bool {
	for _, p := range r.Players {
		if p.Connected && !p.Answered {
			return false
		}
	}
	return r.connectedCountLocked() > 0
}

// AdvanceExpired aplica automaticamente as transições cujo prazo terminou.
// Retorna true quando o estado mudou e precisa ser transmitido aos clientes.
func (r *Room) AdvanceExpired(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.TimerEnd.IsZero() || now.Before(r.TimerEnd) {
		return false
	}

	switch r.Phase {
	case PhaseAnswering:
		for _, p := range r.Players {
			if p.Answered || !p.Connected {
				continue
			}
			p.Answered = true
			r.Answers = append(r.Answers, &AnswerEntry{
				PlayerID:   p.ID,
				PlayerName: p.Name,
				Text:       "(sem resposta)",
				Original:   p.Assigned,
			})
		}
		r.beginDiscussionLocked()
		return true
	case PhaseChanges:
 r.advanceDiscussionLocked()
 return true
 case PhasePower:
		r.applyPowersLocked()
		return true
	default:
		return false
	}
}

func (r *Room) BeginDiscussion() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Phase == PhaseAnswering { r.beginDiscussionLocked() }
}

func (r *Room) beginDiscussionLocked() {
	idx := make([]int, len(r.Answers))
	for i := range idx {
		idx[i] = i
	}
	rand.Shuffle(len(idx), func(i, j int) { idx[i], idx[j] = idx[j], idx[i] })
	r.Discussion = idx
	r.DiscIdx = 0
	r.Phase = PhaseDiscussing
	r.TimerEnd = time.Time{}
}

func (r *Room) CurrentDiscussion() (*AnswerEntry, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.DiscIdx >= len(r.Discussion) {
		return nil, ""
	}
	entry := r.Answers[r.Discussion[r.DiscIdx]]
	return entry, r.judgeForAnswer(entry.PlayerID)
}

func (r *Room) JudgePlace(judgeID string, tier Tier) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.DiscIdx >= len(r.Discussion) {
		return false
	}
	entry := r.Answers[r.Discussion[r.DiscIdx]]
	judge := r.judgeForAnswer(entry.PlayerID)
	if judge != judgeID || r.Phase != PhaseDiscussing || TierIndex(tier) < 0 {
		return false
	}
	entry.Placed = tier
 entry.LeaderName = r.Players[judge].Name
	entry.PlacedSet = true
	for _, p := range r.Players {
		p.PowerChoice = 0
		p.powerChosen = false
	}
	// O autor da resposta em discussão não participa desta decisão de poder.
	// Marcamos como resolvido para não bloquear o avanço antecipado da fase.
	r.Players[entry.PlayerID].powerChosen = true
	r.Phase = PhasePower
	r.TimerEnd = time.Now().Add(30 * time.Second)
	return true
}

func (r *Room) SubmitPower(playerID string, choice int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[playerID]
	if p == nil || !p.Connected || r.Phase != PhasePower || p.powerChosen {
		return false
	}
	if r.DiscIdx >= len(r.Discussion) {
		return false
	}
	entry := r.Answers[r.Discussion[r.DiscIdx]]
	if entry.PlayerID == playerID {
		return false
	}
	if choice < -1 || choice > 1 {
		return false
	}
	if choice != 0 && p.UsedPower {
		return false
	}
	p.PowerChoice = choice
	if choice != 0 {
		p.UsedPower = true
	}
	p.powerChosen = true
	if r.allPowersLocked() { r.applyPowersLocked(); return true }; return false
}

func (r *Room) allPowersLocked() bool {
	for _, p := range r.Players {
		if p.Connected && !p.powerChosen {
			return false
		}
	}
	return true
}

func (r *Room) ApplyPowers() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.applyPowersLocked()
}

func (r *Room) applyPowersLocked() bool {
 if r.Phase != PhasePower || r.DiscIdx >= len(r.Discussion) { return false }
 entry := r.Answers[r.Discussion[r.DiscIdx]]
 before := entry.Placed
 delta := 0
 r.Changes = nil
 for _, id := range r.Order { p := r.Players[id]; if p.PowerChoice != 0 {
 delta -= p.PowerChoice
 r.Changes = append(r.Changes, map[string]any{"playerName":p.Name,"choice":p.PowerChoice})
 } }
 entry.Placed = ShiftTier(before, delta)
 for _, p := range r.Players { p.PowerChoice = 0; p.powerChosen = false }
 if len(r.Changes) > 0 {
 for _, change := range r.Changes { change["before"] = before; change["after"] = entry.Placed }
 r.Phase = PhaseChanges
 r.TimerEnd = time.Now().Add(10*time.Second)
 return true
 }
 return r.advanceDiscussionLocked()
}

func (r *Room) advanceDiscussionLocked() bool {
 r.DiscIdx++
 r.TimerEnd = time.Time{}
 if r.DiscIdx >= len(r.Discussion) { r.finishRoundLocked(); r.Phase=PhaseReview; return false }
 r.Phase=PhaseDiscussing
 return true
}

func (r *Room) Continue(requesterID string) {
 r.mu.Lock(); defer r.mu.Unlock()
 if p:=r.Players[requesterID]; p!=nil && p.Host && r.Phase==PhaseChanges { r.advanceDiscussionLocked() }
}

func (r *Room) finishRoundLocked() {
	score := 0
	for _, a := range r.Answers {
		diff := TierIndex(a.Original) - TierIndex(a.Placed)
		if diff < 0 {
			diff = -diff
		}
		switch {
		case diff == 0:
			score += 3
		case diff == 1:
			score += 1
		default:
			score -= 1
		}
	}
	r.Results = append(r.Results, &RoundResult{
		Theme:   r.CurrentTheme,
		Score:   score,
		Answers: r.Answers,
	})
}

func (r *Room) ChangeTheme(requesterID string) {
 r.mu.Lock(); defer r.mu.Unlock()
 p:=r.Players[requesterID]
 if p==nil || !p.Connected || r.Phase!=PhaseAnswering { return }
 if r.ThemeVotes==nil { r.ThemeVotes=map[string]bool{} }
 r.ThemeVotes[requesterID]=true
 if !p.Host && len(r.ThemeVotes)<=r.connectedCountLocked()/2 { return }
 r.changeThemeLocked()
}
func (r *Room) changeThemeLocked() {
 r.CurrentTheme=r.pickTheme()
 r.UsedThemes=append(r.UsedThemes,r.CurrentTheme)
 r.ThemeVotes=map[string]bool{}
 r.Revision++
 r.Answers=nil
 for _, p:=range r.Players { p.Answer=""; p.Answered=false }
 r.TimerEnd=time.Now().Add(80*time.Second)
}
func (r *Room) connectedCountLocked() int { n:=0; for _,p:=range r.Players { if p.Connected { n++ } }; return n }
func (r *Room) PlayerIDs() []string { r.mu.RLock(); defer r.mu.RUnlock(); return append([]string(nil),r.Order...) }
func (r *Room) Kick(hostID,targetID string) bool {
 r.mu.Lock(); defer r.mu.Unlock()
 h:=r.Players[hostID]
 if h==nil || !h.Host || r.Phase!=PhaseLobby || hostID==targetID || r.Players[targetID]==nil { return false }
 r.removeLobbyPlayerLocked(targetID); return true
}
func (r *Room) removeLobbyPlayerLocked(id string) {
 delete(r.Players,id)
 order:=[]string{}
 for _,pid:=range r.Order { if pid!=id { r.Players[pid].Order=len(order); order=append(order,pid) } }
 r.Order=order
}
func (r *Room) Disconnect(id string) {
 r.mu.Lock(); defer r.mu.Unlock()
 p:=r.Players[id]; if p==nil { return }; wasHost:=p.Host; p.Connected=false; p.Host=false
 delete(r.ThemeVotes,id)
 if r.Phase==PhaseLobby { r.removeLobbyPlayerLocked(id) }
 if wasHost { for _,pid:=range r.Order { if next:=r.Players[pid]; next.Connected { next.Host=true; break } } }
 if r.connectedCountLocked()==0 { return }
 if r.Phase==PhaseAnswering {
 if len(r.ThemeVotes)>r.connectedCountLocked()/2 { r.changeThemeLocked() } else if r.allAnsweredLocked() { r.beginDiscussionLocked() }
 }
 if r.Phase==PhasePower && r.allPowersLocked() { r.applyPowersLocked() }
}

func (r *Room) ViewForPlayer(playerID string) (result map[string]any) {
 r.mu.RLock()
 defer r.mu.RUnlock()
 // Detach slices and answer pointers before the socket serializes the view.
 defer func() { if result!=nil { data,_:=json.Marshal(result); var snapshot map[string]any; json.Unmarshal(data,&snapshot); result=snapshot } }()
	p := r.Players[playerID]
	if p == nil {
		return nil
	}
	view := map[string]any{
		"phase":        string(r.Phase),
		"code":         r.Code,
		"round":        r.Round,
		"currentTheme": r.CurrentTheme,
		"players":      r.playersSummary(),
		"canStart": p.Host && r.connectedCountLocked() >= 3,
 "host": p.Host, "myID": playerID, "revision":r.Revision,
 "themeVotes":len(r.ThemeVotes), "themeVotesNeeded":r.connectedCountLocked()/2+1, "iVotedTheme":r.ThemeVotes[playerID],
	}

	if r.Phase == PhaseLobby {
		return view
	}

	board := []map[string]any{}
 for _,a:=range r.Answers { if a.PlacedSet { board=append(board,map[string]any{"text":a.Text,"playerName":a.PlayerName,"placed":a.Placed}) } }
 view["board"]=board
 view["myLeaderName"]=""
 if leader:=r.Players[r.judgeForAnswer(playerID)]; leader!=nil { view["myLeaderName"]=leader.Name }
 view["myTier"] = string(p.Assigned)
	view["iAnswered"] = p.Answered

	if r.Phase == PhaseAnswering {
		view["timerEnd"] = r.TimerEnd.UnixMilli()
		return view
	}

	if r.Phase == PhaseDiscussing || r.Phase == PhasePower || r.Phase == PhaseChanges {
		if r.DiscIdx < len(r.Discussion) {
			entry := r.Answers[r.Discussion[r.DiscIdx]]
			judge := r.judgeForAnswer(entry.PlayerID)
			view["discussingPlayerID"] = entry.PlayerID
			view["discussingPlayerName"] = entry.PlayerName
			view["discussingText"] = entry.Text
			view["judgeID"] = judge
 if leader:=r.Players[judge]; leader!=nil { view["judgeName"]=leader.Name }
 view["placed"]=entry.Placed
 view["changes"]=r.Changes
 view["myPowerChosen"]=p.powerChosen
			view["iAmJudge"] = playerID == judge
			view["iAmTarget"] = playerID == entry.PlayerID
			view["myUsedPower"] = p.UsedPower
			view["timerEnd"] = r.TimerEnd.UnixMilli()
		}
		return view
	}

	if r.Phase == PhaseReview {
		if len(r.Results) > 0 {
			view["review"] = r.Results[len(r.Results)-1]
		}
		view["nextRound"] = r.Round < 5
		view["host"] = p.Host
		return view
	}

	if r.Phase == PhaseFinal {
		view["results"] = r.Results
		return view
	}

	return view
}

func (r *Room) playersSummary() []map[string]any {
	list := []map[string]any{}
	for _, id := range r.Order {
		p := r.Players[id]
		list = append(list, map[string]any{
			"id":   p.ID,
			"name": p.Name,
			"host": p.Host, "connected":p.Connected,
		})
	}
	return list
}

func ScoreDiff(original, placed Tier) int {
	diff := TierIndex(original) - TierIndex(placed)
	if diff < 0 {
		diff = -diff
	}
	switch diff {
	case 0:
		return 3
	case 1:
		return 1
	default:
		return -1
	}
}
