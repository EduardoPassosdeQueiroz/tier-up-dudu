package game

import (
 "encoding/json"
 "testing"
 "time"
)

func testRoom(t *testing.T) *Room {
 t.Helper()
 r:=NewRoom("TEST")
 r.AddPlayer("h","Host",true); r.AddPlayer("a","Ana",false); r.AddPlayer("b","Bia",false)
 if !r.StartRound("h") {t.Fatal("start failed")}; return r
}
func discussionRoom(t *testing.T) *Room {
 r:=testRoom(t)
 for _,id:=range r.Order {r.SubmitAnswer(id,id+" resposta")}
 r.BeginDiscussion(); return r
}
func TestThemeVotesAndTiming(t *testing.T) {
 r:=testRoom(t); initial:=r.CurrentTheme; rev:=r.Revision
 if left:=time.Until(r.TimerEnd); left<79*time.Second || left>80*time.Second {t.Fatal(left)}
 r.SubmitAnswer("h","rascunho confirmado")
 r.ChangeTheme("a"); r.ChangeTheme("a")
 if r.CurrentTheme!=initial {t.Fatal("duplicate vote changed theme")}
 r.ChangeTheme("b")
 if r.CurrentTheme==initial || r.Revision!=rev+1 || len(r.Answers)!=0 || r.Players["h"].Answered {t.Fatal("majority did not reset theme")}
 previous:=r.CurrentTheme; r.ChangeTheme("h"); if r.CurrentTheme==previous {t.Fatal("host cannot change")}
 r.AdvanceExpired(r.TimerEnd)
 previous=r.CurrentTheme; r.ChangeTheme("h"); if r.CurrentTheme!=previous {t.Fatal("changed outside answering")}
 if len(Themes)!=64 {t.Fatal(len(Themes))}
}
func TestPowersCancelAtEveryTier(t *testing.T) {
 for _,tier:=range TierOrder {t.Run(string(tier),func(t *testing.T){
 r:=discussionRoom(t); entry,judge:=r.CurrentDiscussion()
 if !r.JudgePlace(judge,tier) {t.Fatal("judge failed")}
 if left:=time.Until(r.TimerEnd);left<29*time.Second || left>30*time.Second {t.Fatal(left)}
 choice:=1
 for _,id:=range r.Order {if id!=entry.PlayerID {r.SubmitPower(id,choice);choice=-1}}
 r.ApplyPowers()
 if entry.Placed!=tier || r.Phase!=PhaseChanges || len(r.Changes)!=2 {t.Fatal("opposite powers must cancel",entry.Placed,r.Phase)}
 r.Continue("a"); if r.Phase!=PhaseChanges {t.Fatal("non-host continued")}
 r.Continue("h"); if r.Phase!=PhaseDiscussing {t.Fatal("continue failed")}
 board:=r.ViewForPlayer("h")["board"].([]any)
 if len(board)!=1 || board[0].(map[string]any)["text"]!=entry.Text {t.Fatal("board lost placed answer")}
 })}
}
func TestPowersLimitsAndNoOp(t *testing.T) {
 for _,tc:=range []struct{tier Tier; choice int; want Tier}{{TierS,1,TierS},{TierF,-1,TierF},{TierE,1,TierC},{TierC,-1,TierE},{TierB,0,TierB}} {
 r:=discussionRoom(t); entry,judge:=r.CurrentDiscussion(); r.JudgePlace(judge,tc.tier)
 sent:=false
 for _,id:=range r.Order {if id!=entry.PlayerID {choice:=0;if !sent {choice=tc.choice;sent=true};r.SubmitPower(id,choice)}}
 r.ApplyPowers();if entry.Placed!=tc.want {t.Fatal(tc,entry.Placed)}
 if tc.choice==0 && r.Phase!=PhaseDiscussing {t.Fatal("no-op should skip change screen")}
 if tc.choice!=0 {r.AdvanceExpired(r.TimerEnd);if r.Phase!=PhaseDiscussing {t.Fatal("summary timeout")}}
 }
}
func TestHostKickAndDisconnect(t *testing.T) {
 r:=NewRoom("TEST");r.AddPlayer("h","Host",true);r.AddPlayer("a","Ana",false);r.AddPlayer("b","Bia",false);r.AddPlayer("c","Caio",false)
 if r.Kick("a","b") || r.Kick("h","h") {t.Fatal("unauthorized kick")}
 if !r.Kick("h","b") {t.Fatal("kick failed")};r.Disconnect("h")
 if !r.IsHost("a") || r.Players["c"].Order!=1 {t.Fatal("host order")}
 r.AddPlayer("d","Duda",false);if !r.StartRound("a") {t.Fatal("new host start")}
 if r.Kick("a","c") || r.StartRound("c") || r.StartRound("a") {t.Fatal("phase guards")}
 r.Disconnect("a");if !r.IsHost("c") {t.Fatal("in-game host migration")}
}
func TestSecretAndValidation(t *testing.T) {
 r:=discussionRoom(t);_,judge:=r.CurrentDiscussion()
 if r.JudgePlace(judge,Tier("X")) {t.Fatal("invalid tier accepted")}
 view:=r.ViewForPlayer("h");blob,_:=json.Marshal(view)
 var obj map[string]any;json.Unmarshal(blob,&obj)
 if _,ok:=obj["results"];ok {t.Fatal("results disclosed early")}
 seen:=map[Tier]bool{};for _,p:=range r.Players {if seen[p.Assigned] {t.Fatal("duplicate assignment")};seen[p.Assigned]=true}
}
func TestFiveRoundsReachFinal(t *testing.T) {
 r:=testRoom(t)
 for round:=1;round<=5;round++ {
 r.AdvanceExpired(r.TimerEnd)
 for r.Phase==PhaseDiscussing {_,judge:=r.CurrentDiscussion();r.JudgePlace(judge,TierE);r.AdvanceExpired(r.TimerEnd)}
 if r.Phase!=PhaseReview {t.Fatal(r.Phase)}
 r.StartRound("h")
 }
 if r.Phase!=PhaseFinal || len(r.Results)!=5 {t.Fatal("final inaccessible")}
}
