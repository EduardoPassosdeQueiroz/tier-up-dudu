package game

import (
	"testing"
	"time"
)

func TestScoreEveryDistance(t *testing.T) {
	want := []int{3, 1, -1, -3, -5, -7}
	for i, tier := range TierOrder {
		if got := ScoreDiff(TierS, tier); got != want[i] { t.Fatalf("distance %d: got %d want %d", i, got, want[i]) }
		if got := ScoreDiff(tier, TierS); got != want[i] { t.Fatalf("asymmetric score: %d", got) }
	}
}

func TestInstructionsRequireEveryConfirmation(t *testing.T) {
	r := NewRoom("TEST")
	for _, id := range []string{"h", "a", "b"} { r.AddPlayer(id, id, id == "h") }
	r.StartRound("h")
	if r.Phase != PhaseInstructions || r.Round != 0 || !r.TimerEnd.IsZero() { t.Fatal("game started before instructions") }
	r.ReadyToPlay("h"); r.ReadyToPlay("h"); r.ReadyToPlay("a")
	if r.Phase != PhaseInstructions { t.Fatal("duplicate confirmation counted twice") }
	r.Disconnect("b")
	if r.Phase != PhaseInstructions { t.Fatal("disconnect bypassed confirmation") }
	r.SetSession("b", "b-token"); r.Resume("b-token")
	if !r.ReadyToPlay("b") || r.Phase != PhaseVoting || r.Round != 1 { t.Fatal("last confirmation failed to start") }
	chooseTheme(t,r)
	if remaining:=time.Until(r.TimerEnd); remaining<119*time.Second || remaining>120*time.Second { t.Fatal(remaining) }
	if r.ReadyToPlay("b") { t.Fatal("ready accepted outside instructions") }
}

func TestAnswerEditRevokesConfirmationAndRetainsDeadline(t *testing.T) {
	r := testRoom(t)
	deadline := r.TimerEnd
	r.SubmitAnswer("h", "Primeira versão")
	if !r.EditAnswer("h") || r.Players["h"].Answered || len(r.Answers) != 0 || !r.TimerEnd.Equal(deadline) { t.Fatal("edit did not revoke confirmation") }
	r.SubmitAnswer("a", "Outra resposta"); r.SubmitAnswer("b", "Mais uma")
	if r.Phase != PhaseAnswering { t.Fatal("advanced while author was editing") }
	r.SubmitAnswer("h", "Versão corrigida")
	if r.Phase != PhaseDiscussing || len(r.Answers) != 3 { t.Fatal("did not advance after all reconfirmed") }
	for _, a := range r.Answers { if a.PlayerID == "h" && a.Text != "Versão corrigida" { t.Fatal("stale answer") } }
	if r.EditAnswer("h") { t.Fatal("edited after discussion began") }
}

func TestEditingAtDeadlineUsesLastSentText(t *testing.T) {
	r:=testRoom(t)
	r.SubmitAnswer("h", "Última resposta enviada"); r.EditAnswer("h")
	r.AdvanceExpired(r.TimerEnd)
	for _,a:=range r.Answers {if a.PlayerID=="h" && a.Text!="Última resposta enviada" {t.Fatal("deadline lost the sent text")}}
}

func TestLikesArePerPlayerAndCannotLikeOwnAnswer(t *testing.T) {
	r:=testRoom(t)
	r.Results=[]*RoundResult{{Theme:"Teste",Answers:[]*AnswerEntry{{PlayerID:"h",PlayerName:"Host",Text:"Resposta",Original:TierA,Placed:TierA}}}}
	if r.ToggleLike("a",1,"h") {t.Fatal("like accepted before results")}
	r.Phase=PhaseReview
	if r.ToggleLike("h",1,"h") || r.ToggleLike("missing",1,"h") || r.ToggleLike("a",0,"h") || r.ToggleLike("a",2,"h") {t.Fatal("invalid like accepted")}
	r.ToggleLike("a",1,"h");r.ToggleLike("b",1,"h")
	a:=r.Results[0].Answers[0]
	if len(a.Likes)!=2 {t.Fatal("likes not counted")}
	r.ToggleLike("a",1,"h")
	if len(a.Likes)!=1 || a.Likes["a"] {t.Fatal("like not removed")}
	r.Phase=PhaseFinal
	r.ToggleLike("a",1,"h")
	if len(a.Likes)!=2 {t.Fatal("final likes failed")}
}

func TestPowerContributionIgnoresOtherPowers(t *testing.T) {
	r:=NewRoom("TEST")
	for _,id:=range []string{"h","a","b","c"} {r.AddPlayer(id,id,id=="h")}
	r.StartRound("h");for _,id:=range r.Order {r.ReadyToPlay(id)}
	entry:=&AnswerEntry{PlayerID:"h",PlayerName:"h",Original:TierB,Placed:TierC,PlacedSet:true}
	r.Answers=[]*AnswerEntry{entry};r.Discussion=[]int{0};r.Phase=PhasePower;r.Players["h"].powerChosen=true
	r.SubmitPower("a",1);r.SubmitPower("b",-1);r.SubmitPower("c",-1)
	if entry.Placed!=TierE || ScoreDiff(entry.Original,entry.Placed)!=-1 {t.Fatal("wrong net team outcome")}
	points:=map[string]int{};for _,action:=range entry.Powers {points[action.PlayerID]+=action.Points}
	if points["a"]!=2 || points["b"]!=-2 || points["c"]!=-2 {t.Fatal("individual contribution used net outcome",points)}
}

func TestPowerStatsAccumulateAcrossThemes(t *testing.T) {
	r:=testRoom(t)
	r.Results=[]*RoundResult{
		{Answers:[]*AnswerEntry{{PlayerID:"h",PlayerName:"h",Original:TierA,Placed:TierA,Powers:[]PowerAction{{PlayerID:"a",PlayerName:"a",Points:2}}}}},
		{Answers:[]*AnswerEntry{{PlayerID:"h",PlayerName:"h",Original:TierB,Placed:TierB,Powers:[]PowerAction{{PlayerID:"a",PlayerName:"a",Points:2}}}}},
		{Answers:[]*AnswerEntry{{PlayerID:"h",PlayerName:"h",Original:TierC,Placed:TierE,Powers:[]PowerAction{{PlayerID:"a",PlayerName:"a",Points:-2}}}}},
	}
	for _,stat:=range r.statsLocked() {if stat["playerId"]=="a" && (stat["powerScore"]!=2 || stat["powerUses"]!=3) {t.Fatal(stat)}}
}

func TestIndependentTierDrawsPermitRepeatsWithSixPlayers(t *testing.T) {
	r:=NewRoom("TEST")
	for _,id:=range []string{"a","b","c","d","e","f"} {r.AddPlayer(id,id,id=="a")}
	// A deck that distributes all six distinct letters every time fails this check.
	for attempt:=0;attempt<64;attempt++ {
		r.assignTiers();seen:=map[Tier]bool{}
		for _,p:=range r.Players {if seen[p.Assigned] {return};seen[p.Assigned]=true}
	}
	t.Fatal("draws appear to enforce unique letters")
}

func TestLeavingInvalidatesSession(t *testing.T) {
 r:=testRoom(t)
 r.SetSession("h","private-token");r.Leave("h")
 if id,_:=r.Resume("private-token");id!="" {t.Fatal("left session resumed")}
 if !r.IsHost("a") {t.Fatal("host not transferred on leave")}
}
