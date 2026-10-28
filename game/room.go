package game

import (
	"math/rand"
	"sync"
	"time"
)

type Tier string

const (
	TierS Tier = "S"
	TierA Tier = "A"
	TierB Tier = "B"
	TierC Tier = "C"
	TierF Tier = "F"
)

var TierOrder = []Tier{TierS, TierA, TierB, TierC, TierF}

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
	p := &Player{
		ID:        id,
		Name:      name,
		Host:      host,
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
	next := (owner.Order + 1) % len(r.Order)
	return r.Order[next]
}

func (r *Room) assignTiers() {
	tiers := []Tier{TierS, TierA, TierB, TierC, TierF}
	for _, p := range r.Players {
		p.Assigned = tiers[rand.Intn(len(tiers))]
		p.Answer = ""
		p.Answered = false
		p.UsedPower = false
		p.PowerChoice = 0
		p.powerChosen = false
	}
}

var Themes = []string{
	"Comidas típicas de Natal", "Frutas tropicais", "Doces de festa infantil",
	"Comidas de festa junina", "Pratos da culinária japonesa", "Sabores de pizza",
	"Recheios de pastel", "Bebidas", "Sobremesas com chocolate",
	"Animais marinhos", "Animais de fazenda", "Aves que não voam",
	"Animais que vivem na África", "Animais extintos", "Animais que vivem no gelo",
	"Insetos que voam", "Animais que vivem em rios", "Animais que possuem chifres",
	"Vilões da Disney", "Personagens da Marvel/DC", "Youtubers",
	"Personagens que dão mais medo em filmes", "Séries", "Personagens de Harry Potter",
	"Personagens de séries de TV aberta", "Atores", "Filmes de animação",
	"Países da Europa", "Lugares para se esconder num esconde-esconde mortal",
	"Lugares para passar a lua de mel", "Lugares para levar alguém num primeiro encontro horrível",
	"Pontos turísticos famosos", "Lugares para pedir alguém em casamento",
	"Países da África", "Lugares para você dormir", "Lugares onde faz muito frio",
	"Personagens de Clash Royale", "Mobs de Minecraft",
	"Marcas de eletrônicos", "Jogos de tabuleiro", "Jogos online",
	"Personagens fictícios que seriam presos na vida real",
	"Coisas que você levaria para sobreviver numa ilha deserta",
	"Coisas suspeitas de encontrar no porta-malas de um carro",
	"Objetos valiosos após o fim do mundo",
	"Coisas que você encontra debaixo da cama", "Desculpas para chegar atrasado",
	"Coisas que não se deve falar num casamento", "Motivos para terminar um namoro",
	"Coisas que irritam professores", "Objetos que podem machucar se caírem na cabeça",
	"Presentes ruins de aniversário", "Coisas que dão medo durante a madrugada",
	"Coisas que você não deveria pesquisar no Google",
	"Jogadores brasileiros de futebol", "Times do Brasil", "Esportes olímpicos",
	"Atletas que você não deixaria cuidar do seu filho", "Atletas que venceriam uma briga de bar",
	"Bandas", "Maiores atletas da história", "Celebridades que seriam ótimos vilões",
	"Cantoras pop internacionais",
	"Super-poderes", "Dinossauros", "Invenções que não deveriam existir",
	"Invenções que mudaram o mundo", "Órgãos do corpo humano",
	"Objetos atuais que assustariam pessoas da Idade Média",
	"O que você faria se pudesse parar o tempo", "Coisas que faria se pudesse viajar no tempo",
	"Acontecimentos históricos",
	"Marcas de carros", "Objetos de cozinha mais úteis", "Materiais escolares",
	"É trair ou não", "Marcas de tênis", "Mínimo ou tratamento de princesa",
	"Conhecidos atraentes", "Green flags e red flags", "Vergonha ou motivo de orgulho",
	"Personagens que usam máscara", "Coisas que brilham no escuro",
	"Profissões que utilizam uniformes", "Personagens fictícios com poderes de gelo",
	"Objetos que podem ser encontrados no fundo do oceano", "Coisas proibidas em aviões",
	"Personagens que morreram em filmes", "Coisas que custam mais de R$ 1 milhão",
	"Objetos que existiam antes da internet", "Coisas úteis num apocalipse zumbi",
	"Personagens fictícios que perderam alguma parte do corpo",
	"Coisas que podem causar o fim da humanidade", "Personagens que traíram aliados",
	"Objetos que poderiam ser armas improvisadas",
	"Coisas encontradas numa casa abandonada",
	"Personagens com identidade secreta",
	"Coisas que um bilionário compraria por diversão",
	"Coisas proibidas numa sociedade futurista",
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
		return Themes[rand.Intn(len(Themes))]
	}
	return avail[rand.Intn(len(avail))]
}

func (r *Room) StartRound() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Round >= 5 {
		r.Phase = PhaseFinal
		return false
	}
	r.Round++
	theme := r.pickTheme()
	r.UsedThemes = append(r.UsedThemes, theme)
	r.CurrentTheme = theme
	r.Answers = nil
	r.Discussion = nil
	r.DiscIdx = 0
	r.assignTiers()
	r.Phase = PhaseAnswering
	r.TimerEnd = time.Now().Add(60 * time.Second)
	return true
}

func (r *Room) SubmitAnswer(playerID, text string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[playerID]
	if p == nil || p.Answered || r.Phase != PhaseAnswering {
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
	return r.allAnsweredLocked()
}

func (r *Room) allAnsweredLocked() bool {
	for _, p := range r.Players {
		if !p.Answered {
			return false
		}
	}
	return len(r.Players) >= 3
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
			if p.Answered {
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
	r.beginDiscussionLocked()
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
	if judge != judgeID || r.Phase != PhaseDiscussing {
		return false
	}
	entry.Placed = tier
	entry.PlacedSet = true
	for _, p := range r.Players {
		p.PowerChoice = 0
		p.powerChosen = false
	}
	// O autor da resposta em discussão não participa desta decisão de poder.
	// Marcamos como resolvido para não bloquear o avanço antecipado da fase.
	r.Players[entry.PlayerID].powerChosen = true
	r.Phase = PhasePower
	r.TimerEnd = time.Now().Add(25 * time.Second)
	return true
}

func (r *Room) SubmitPower(playerID string, choice int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.Players[playerID]
	if p == nil || r.Phase != PhasePower || p.powerChosen {
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
	return r.allPowersLocked()
}

func (r *Room) allPowersLocked() bool {
	for _, p := range r.Players {
		if !p.powerChosen {
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
	if r.DiscIdx < len(r.Discussion) {
		entry := r.Answers[r.Discussion[r.DiscIdx]]
		if entry.PlacedSet {
			delta := 0
			for _, p := range r.Players {
				if p.PowerChoice == 1 {
					delta--
				} else if p.PowerChoice == -1 {
					delta++
				}
			}
			if delta != 0 {
				entry.Placed = ShiftTier(entry.Placed, delta)
			}
		}
	}
	for _, p := range r.Players {
		p.PowerChoice = 0
		p.powerChosen = false
	}
	r.DiscIdx++
	if r.DiscIdx >= len(r.Discussion) {
		r.finishRoundLocked()
		r.Phase = PhaseReview
		r.TimerEnd = time.Time{}
		return false
	}
	r.Phase = PhaseDiscussing
	r.TimerEnd = time.Time{}
	return true
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
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Phase == PhaseLobby || r.Phase == PhaseFinal {
		return
	}
	p := r.Players[requesterID]
	if p == nil {
		return
	}
	if !p.Host {
		if r.Phase == PhaseDiscussing || r.Phase == PhasePower {
			if r.DiscIdx >= len(r.Discussion) {
				return
			}
			entry := r.Answers[r.Discussion[r.DiscIdx]]
			if entry.PlayerID != requesterID {
				return
			}
		} else {
			return
		}
	}
	newTheme := r.pickTheme()
	if len(r.UsedThemes) > 0 {
		r.UsedThemes[len(r.UsedThemes)-1] = newTheme
	} else {
		r.UsedThemes = append(r.UsedThemes, newTheme)
	}
	r.CurrentTheme = newTheme
	r.Answers = nil
	r.Discussion = nil
	r.DiscIdx = 0
	r.assignTiers()
	r.Phase = PhaseAnswering
	r.TimerEnd = time.Now().Add(60 * time.Second)
}

func (r *Room) ViewForPlayer(playerID string) map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
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
		"canStart":     p.Host && len(r.Players) >= 3,
	}

	if r.Phase == PhaseLobby {
		return view
	}

	view["myTier"] = string(p.Assigned)
	view["iAnswered"] = p.Answered

	if r.Phase == PhaseAnswering {
		view["timerEnd"] = r.TimerEnd.UnixMilli()
		return view
	}

	if r.Phase == PhaseDiscussing || r.Phase == PhasePower {
		if r.DiscIdx < len(r.Discussion) {
			entry := r.Answers[r.Discussion[r.DiscIdx]]
			judge := r.judgeForAnswer(entry.PlayerID)
			view["discussingPlayerID"] = entry.PlayerID
			view["discussingPlayerName"] = entry.PlayerName
			view["discussingText"] = entry.Text
			view["judgeID"] = judge
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
			"host": p.Host,
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
