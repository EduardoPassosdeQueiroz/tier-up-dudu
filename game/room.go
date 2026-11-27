package game

import (
	"encoding/json"
	"math/rand"
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
    PhaseLobby        Phase = "lobby"
    PhaseInstructions Phase = "instructions"
    PhaseThemeVote    Phase = "themeVote"
    PhaseAnswering    Phase = "answering"
    PhaseDiscussing   Phase = "discussing"
    PhasePower        Phase = "power"
    PhaseChanges      Phase = "changes"
    PhaseReview       Phase = "review"
    PhaseFinal        Phase = "final"
)

type Player struct {
	SessionToken string `json:"-"`
	ID           string `json:"id"`
	Name         string `json:"name"`
	Host         bool   `json:"host"`
	Order        int    `json:"order"`
	Connected    bool   `json:"connected"`

	Assigned    Tier   `json:"-"`
	Answer      string `json:"-"`
	Answered    bool   `json:"answered"`
	UsedPower   bool   `json:"usedPower"`
	PowerChoice int    `json:"-"`
	powerChosen bool   `json:"-"`
	Ready       bool   `json:"-"`
	Genius      int    `json:"-"`
}

type AnswerEntry struct {
	Likes      map[string]bool `json:"likes"`
	LeaderName string          `json:"leaderName,omitempty"`
	PlayerID   string          `json:"playerId"`
	PlayerName string          `json:"playerName"`
	Text       string          `json:"text"`
	Original   Tier            `json:"original,omitempty"`
	Placed     Tier            `json:"placed,omitempty"`
	PlacedSet  bool            `json:"placedSet"`
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

	// Os 15 temas exclusivos da partida.
	ThemeDeck []string

	// Cinco opções apresentadas na rodada atual.
	ThemeOptions []string

	// ID do jogador -> tema escolhido.
	ThemeVotes map[string]string

	Answers    []*AnswerEntry
	Discussion []int
	DiscIdx    int

	TimerEnd time.Time

	Revision int
	Changes  []map[string]any
	Results  []*RoundResult
}

func NewRoom(code string) *Room {
	return &Room{
		Code:    code,
		Players: map[string]*Player{},
		Phase:   PhaseLobby,
	}
}

func (r *Room) AddPlayer(id, name string, host bool) *Player {
	p, _ := r.AddPlayerChecked(id, name, host)
	return p
}

func (r *Room) AddPlayerChecked(id, name string, host bool) (*Player, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Phase != PhaseLobby {
		return nil, "partida_em_andamento"
	}
	for _, existing := range r.Players {
		if strings.EqualFold(strings.TrimSpace(existing.Name), strings.TrimSpace(name)) {
			return nil, "nome_duplicado"
		}
	}
	p := &Player{
		ID:        id,
		Name:      name,
		Host:      host || r.connectedCountLocked() == 0,
		Order:     len(r.Order),
		Connected: true,
	}
	r.Players[id] = p
	r.Order = append(r.Order, id)
	return p, ""
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
		id := r.Order[(owner.Order+offset)%len(r.Order)]
		if p := r.Players[id]; p != nil && p.Connected {
			return id
		}
	}
	return ""
}

func (r *Room) assignTiers() {

	for _, p := range r.Players {
		p.Assigned = TierOrder[rand.Intn(len(TierOrder))]
		p.Answer = ""
		p.Answered = false
		p.UsedPower = false
		p.PowerChoice = 0
		p.powerChosen = false
	}
}

var Themes = []string{
    "Comidas de feriados (Natal, Festa Junina, etc)",
    "Frutas",
    "Sabores de pizza",
    "Bebidas",
    "Sobremesas",
    "Um animal para ter como pet",
    "Animais extintos",
    "Insetos",
    "Vilões",
    "Ele(a) é um 10 mas...",
    "Tóxico em uma relação (S) ou normal (F)",
    "Personagens da Marvel/DC",
    "Youtubers",
    "Séries de streamings",
    "Personagens com mais aura",
    "Personagens de séries de TV Aberta",
    "Filmes de animação",
    "Lugares para se esconder num esconde-esconde mortal",
    "Lugares para passar a lua de mel",
    "Lugares para levar alguém em um primeiro encontro",
    "Melhores pontos turísticos",
    "Lugar para você dormir",
    "Personagens de Clash Royale",
    "Melhores Mobs de Minecraft",
    "Jogos de tabuleiro",
    "Jogos On-line ou de Videogame",
    "Coisas que você levaria para sobreviver em uma ilha deserta",
    "Coisas que seriam suspeitas/estranhas de encontrar no porta-malas de um carro",
    "Objetos que seriam valiosos após o fim do mundo",
    "Desculpas para chegar atrasado (S - Válida ou Absurda / F - Impossível de acreditar ou Muito básica)",
    "Coisas que não se deve falar em um casamento sendo um noivo(a)",
    "Objetos que podem machucar se caírem na sua cabeça",
    "Presentes de aniversário (sua opinião)",
    "O que você não deveria pesquisar no Google na conta da sua mãe",
    "Jogadores de futebol (hoje em dia)",
    "Maiores times do Brasil",
    "Melhores esportes olímpicos de se jogar",
    "Alguém que você deixaria cuidar do seu filho",
    "Você ganharia desse cara numa luta?",
    "Bandas e Cantores",
    "Super-poderes",
    "Invenções que não deveriam existir",
    "Invenções que vocês gostariam que fossem suas",
    "Órgãos do corpo humano",
    "Objetos atuais que assustariam pessoas da Idade da Pedra",
    "O que você faria se pudesse parar o tempo por 1 mês",
    "Coisas que você faria se pudesse viajar no tempo",
    "Objetos/Eletrodomésticos de cozinha mais úteis",
    "Materiais escolares mais úteis",
    "Termina (S) ou dá uma segunda chance (F)",
    "Tratamento de princesa (S) ou mínimo (F)",
    "Conhecidos bonitos (quem namora não discute)",
    "Red Flag (S) e Green Flag (F)",
    "Motivo de orgulho (S) ou vergonha (F)",
    "Melhores profissões (desconsidere salário)",
    "Melhores coisas que custam mais de R$ 1 milhão",
    "Coisas que seriam úteis em um apocalipse zumbi",
    "Coisas que mais poderiam causar o fim da humanidade",
    "Personagens de animação mais atraentes",
    "Comidas de café da manhã",
    "Sabores de sorvete",
    "Trends da história do tiktok",
    "Atitudes atraentes",
    "Celebridade (S) ou sub-celebridades (F)",
    "Como você terminaria com alguém sem machucar essa pessoa",
    "Melhores emojis",
    "Animais que você ganharia na luta",
    "Melhor fazer sozinho (S) ou melhor fazer com outras pessoas (F)",
    "Animais que você mais gostaria de ser",
    "Dia da semana e horário (4 da tarde de uma terça)",
    "Exige sorte (S) ou exige habilidade (F)",
    "Comida típica do brasil",
    "Imperdoável (S) ou Perdoável (F)",
    "Uma idade ideal que você gostaria de ver seu filho namorando",
}

func (r *Room) prepareThemeDeckLocked() {
    indexes := rand.Perm(len(Themes))

    r.ThemeDeck = make([]string, 15)

    for i := 0; i < 15; i++ {
        r.ThemeDeck[i] = Themes[indexes[i]]
    }

    r.UsedThemes = nil
}

func (r *Room) beginThemeVoteLocked() {
    start := (r.Round - 1) * 5
    end := start + 5

    r.ThemeOptions = append(
        []string(nil),
        r.ThemeDeck[start:end]...,
    )

    // Todas as cinco sugestões ficam indisponíveis
    // nas próximas rodadas.
    r.UsedThemes = append(
        r.UsedThemes,
        r.ThemeOptions...,
    )

    r.ThemeVotes = make(map[string]string)
    r.CurrentTheme = ""
    r.Answers = nil
    r.Discussion = nil
    r.DiscIdx = 0
    r.Changes = nil

    r.Revision++
    r.Phase = PhaseThemeVote
    r.TimerEnd = time.Time{}
}

func (r *Room) SubmitThemeVote(playerID, theme string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    p := r.Players[playerID]

    if p == nil || !p.Connected || r.Phase != PhaseThemeVote {
        return false
    }

    if _, voted := r.ThemeVotes[playerID]; voted {
        return false
    }

    valid := false

    for _, option := range r.ThemeOptions {
        if option == theme {
            valid = true
            break
        }
    }

    if !valid {
        return false
    }

    r.ThemeVotes[playerID] = theme

    if len(r.ThemeVotes) == r.connectedCountLocked() {
        r.resolveThemeVoteLocked()
        return true
    }

    return false
}

func (r *Room) resolveThemeVoteLocked() {
    counts := make(map[string]int)

    for _, theme := range r.ThemeVotes {
        counts[theme]++
    }

    maxVotes := 0
    winners := []string{}

    for _, theme := range r.ThemeOptions {
        votes := counts[theme]

        if votes > maxVotes {
            maxVotes = votes
            winners = []string{theme}
        } else if votes == maxVotes {
            winners = append(winners, theme)
        }
    }

    if len(winners) == 0 {
        return
    }

    r.CurrentTheme = winners[rand.Intn(len(winners))]

    r.assignTiers()
    r.Phase = PhaseAnswering
    r.TimerEnd = time.Now().Add(120 * time.Second)
    r.Revision++
}

func (r *Room) StartRound(requesterID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p := r.Players[requesterID]; p == nil || !p.Host || (r.Phase != PhaseLobby && r.Phase != PhaseReview) {
		return false
	}
	if r.Phase == PhaseLobby && r.connectedCountLocked() < 3 {
		return false
	}
	if r.Phase == PhaseLobby {
		r.Phase = PhaseInstructions
		for _, p := range r.Players {
			p.Ready = false
		}
		return true
	}
	if r.Round >= 3 {
    r.Phase = PhaseFinal
    return false
	}
	r.Round++
	r.beginThemeVoteLocked()
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
	if text == "" || len([]rune(text)) > 100 {
		return false
	}
	p.Answer = text
	p.Answered = true
	r.Answers = append(r.Answers, &AnswerEntry{
		PlayerID:   p.ID,
		PlayerName: p.Name,
		Text:       text,
		Original:   p.Assigned,
	})
	if r.allAnsweredLocked() {
		r.beginDiscussionLocked()
		return true
	}
	return false
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
	if r.Phase == PhaseAnswering {
		r.beginDiscussionLocked()
	}
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

    	// Quem já usou o poder nesta rodada
    	// não precisa decidir novamente.
    	p.powerChosen = p.UsedPower
}

	// O autor da resposta não participa da decisão.
	r.Players[entry.PlayerID].powerChosen = true
	if r.allPowersLocked() {
    	r.applyPowersLocked()
	}
	r.Phase = PhasePower
	r.TimerEnd = time.Now().Add(15 * time.Second)
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
	if r.allPowersLocked() {
		r.applyPowersLocked()
		return true
	}
	return false
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
	if r.Phase != PhasePower || r.DiscIdx >= len(r.Discussion) {
		return false
	}
	entry := r.Answers[r.Discussion[r.DiscIdx]]
	before := entry.Placed
	delta := 0
	for _, p := range r.Players {
		if p.PowerChoice != 0 {
			p.Genius += ScoreDiff(entry.Original, ShiftTier(before, -p.PowerChoice)) - ScoreDiff(entry.Original, before)
		}
	}
	r.Changes = nil
	for _, id := range r.Order {
		p := r.Players[id]
		if p.PowerChoice != 0 {
			delta -= p.PowerChoice
			r.Changes = append(r.Changes, map[string]any{"playerName": p.Name, "choice": p.PowerChoice})
		}
	}
	entry.Placed = ShiftTier(before, delta)
	for _, p := range r.Players {
		p.PowerChoice = 0
		p.powerChosen = false
	}
	if len(r.Changes) > 0 {
		for _, change := range r.Changes {
			change["before"] = before
			change["after"] = entry.Placed
		}
		r.Phase = PhaseChanges
		r.TimerEnd = time.Now().Add(10 * time.Second)
		return true
	}
	return r.advanceDiscussionLocked()
}

func (r *Room) advanceDiscussionLocked() bool {
	r.DiscIdx++
	r.TimerEnd = time.Time{}
	if r.DiscIdx >= len(r.Discussion) {
		r.finishRoundLocked()
		r.Phase = PhaseReview
		return false
	}
	r.Phase = PhaseDiscussing
	return true
}

func (r *Room) Continue(requesterID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p := r.Players[requesterID]; p != nil && p.Host && r.Phase == PhaseChanges {
		r.advanceDiscussionLocked()
	}
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
			score += 3 - 2*diff
		}
	}
	r.Results = append(r.Results, &RoundResult{
		Theme:   r.CurrentTheme,
		Score:   score,
		Answers: r.Answers,
	})
}

func (r *Room) connectedCountLocked() int {
	n := 0
	for _, p := range r.Players {
		if p.Connected {
			n++
		}
	}
	return n
}
func (r *Room) PlayerIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.Order...)
}
func (r *Room) Kick(hostID, targetID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := r.Players[hostID]
	if h == nil || !h.Host || r.Phase != PhaseLobby || hostID == targetID || r.Players[targetID] == nil {
		return false
	}
	r.removeLobbyPlayerLocked(targetID)
	return true
}
func (r *Room) removeLobbyPlayerLocked(id string) {
	delete(r.Players, id)
	order := []string{}
	for _, pid := range r.Order {
		if pid != id {
			r.Players[pid].Order = len(order)
			order = append(order, pid)
		}
	}
	r.Order = order
}
func (r *Room) Disconnect(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[id]
	if p == nil {
		return
	}
	wasHost := p.Host
	p.Connected = false
	p.Host = false
	delete(r.ThemeVotes, id)
	if wasHost {
		for _, pid := range r.Order {
			if next := r.Players[pid]; next.Connected {
				next.Host = true
				break
			}
		}
	}
	if r.connectedCountLocked() == 0 {
		return
	}
	if r.Phase == PhaseInstructions && r.readyCountLocked() == r.connectedCountLocked() {
		r.startFirstRoundLocked()
	}
	if r.Phase == PhaseThemeVote && len(r.ThemeVotes) == r.connectedCountLocked() {
		r.resolveThemeVoteLocked()
	}
	if r.Phase == PhaseAnswering && r.allAnsweredLocked() {
		r.beginDiscussionLocked()
	}
	if r.Phase == PhasePower && r.allPowersLocked() {
		r.applyPowersLocked()
	}
}

func (r *Room) ViewForPlayer(playerID string) (result map[string]any) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Detach slices and answer pointers before the socket serializes the view.
	defer func() {
		if result != nil {
			data, _ := json.Marshal(result)
			var snapshot map[string]any
			json.Unmarshal(data, &snapshot)
			result = snapshot
		}
	}()
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
		"canStart":     p.Host && r.connectedCountLocked() >= 3,
		"ready":        p.Ready,
		"host":         p.Host, "myID": playerID, "revision": r.Revision,
		"themeOptions":     r.ThemeOptions,
		"themeVoteCounts":  r.themeVoteCountsLocked(),
		"myThemeVote":      r.ThemeVotes[playerID],
		"themeVotes":       len(r.ThemeVotes),
		"themeVotesNeeded": r.connectedCountLocked(),
	}

	// Durante a votação de tema, o jogador recebe apenas as opções e a contagem
	// de votos. Nada de respostas, tabuleiro ou informações de rodada.
	if r.Phase == PhaseThemeVote {
		return view
	}

	if r.Phase == PhaseLobby || r.Phase == PhaseInstructions {
		view["readyCount"] = r.readyCountLocked()
		view["connectedCount"] = r.connectedCountLocked()
		return view
	}

	board := []map[string]any{}
	for _, a := range r.Answers {
		if a.PlacedSet {
			board = append(board, map[string]any{"text": a.Text, "playerName": a.PlayerName, "placed": a.Placed})
		}
	}
	view["board"] = board
	view["myLeaderName"] = ""
	if leader := r.Players[r.judgeForAnswer(playerID)]; leader != nil {
		view["myLeaderName"] = leader.Name
	}
	view["myTier"] = string(p.Assigned)
	view["iAnswered"] = p.Answered
	view["myAnswer"] = p.Answer

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
			if leader := r.Players[judge]; leader != nil {
				view["judgeName"] = leader.Name
			}
			view["placed"] = entry.Placed
			view["changes"] = r.Changes
			view["myPowerChosen"] = p.powerChosen
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
		view["nextRound"] = r.Round < 3
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
			"host": p.Host, "connected": p.Connected, "genius": p.Genius,
		})
	}
	return list
}

func (r *Room) themeVoteCountsLocked() map[string]int {
	counts := make(map[string]int)

	for _, theme := range r.ThemeVotes {
		counts[theme]++
	}

	return counts
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
		return 3 - 2*diff
	}
}

// Ready starts the first round once every connected player has acknowledged the rules.
func (r *Room) Ready(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p := r.Players[id]; p != nil && p.Connected && r.Phase == PhaseInstructions {
		p.Ready = true
	}
	if r.Phase == PhaseInstructions && r.connectedCountLocked() > 0 && r.readyCountLocked() == r.connectedCountLocked() {
		r.startFirstRoundLocked()
	}
}
func (r *Room) startFirstRoundLocked() {
    r.Round = 1
    r.prepareThemeDeckLocked()
    r.beginThemeVoteLocked()
}
func (r *Room) readyCountLocked() int {
	n := 0
	for _, p := range r.Players {
		if p.Connected && p.Ready {
			n++
		}
	}
	return n
}
func (r *Room) EditAnswer(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[id]
	if p == nil || !p.Connected || r.Phase != PhaseAnswering || !p.Answered {
		return
	}
	p.Answered = false
	for i, a := range r.Answers {
		if a.PlayerID == id {
			r.Answers = append(r.Answers[:i], r.Answers[i+1:]...)
			break
		}
	}
}
func (r *Room) Like(id string, round int, owner string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[id]
	if p == nil || !p.Connected || (r.Phase != PhaseReview && r.Phase != PhaseFinal) || round < 0 || round >= len(r.Results) || owner == id {
		return
	}
	for _, a := range r.Results[round].Answers {
		if a.PlayerID == owner {
			if a.Likes == nil {
				a.Likes = map[string]bool{}
			}
			if a.Likes[id] {
				delete(a.Likes, id)
			} else {
				a.Likes[id] = true
			}
			return
		}
	}
}
func (r *Room) Leave(id string) {
	r.mu.Lock()
	p := r.Players[id]
	if p == nil {
		r.mu.Unlock()
		return
	}
	p.SessionToken = ""
	if r.Phase == PhaseLobby {
		r.removeLobbyPlayerLocked(id)
		if p.Host {
			for _, pid := range r.Order {
				if q := r.Players[pid]; q.Connected {
					q.Host = true
					break
				}
			}
		}
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()
	r.Disconnect(id)
}

func (r *Room) ReturnToLobby(requesterID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    p := r.Players[requesterID]

    if p == nil || !p.Connected || !p.Host || r.Phase != PhaseFinal {
        return false
    }

    r.Phase = PhaseLobby

    r.Round = 0
    r.Revision++

    r.UsedThemes = nil
    r.ThemeDeck = nil
    r.ThemeOptions = nil
    r.CurrentTheme = ""

    r.ThemeVotes = nil
    r.Answers = nil
    r.Discussion = nil
    r.DiscIdx = 0

    r.TimerEnd = time.Time{}
    r.Changes = nil
    r.Results = nil

    for _, player := range r.Players {
        player.Assigned = ""
        player.Answer = ""
        player.Answered = false
        player.UsedPower = false
        player.PowerChoice = 0
        player.powerChosen = false
        player.Ready = false
        player.Genius = 0
    }

    return true
}
