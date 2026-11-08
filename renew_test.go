package main

import (
 "net/http"
 "os"
 "path/filepath"
 "testing"
 "time"

 "github.com/tidwall/gjson"
)

func TestRenewProtocolAndMinimumOutput(t *testing.T) {
 for _,tc:=range []struct{format,body,path,cap string}{
  {"claude",`{"model":"claude-opus","max_tokens":500,"thinking":{"type":"adaptive"},"messages":[{"role":"user","content":"Hi"}]}`,"/v1/messages","max_tokens"},
  {"openai-response",`{"model":"claude-opus","input":"Hi","max_output_tokens":500}`,"/v1/responses","max_output_tokens"},
  {"openai",`{"model":"claude-opus","messages":[{"role":"user","content":"Hi"}]}`,"/v1/chat/completions","max_tokens"},
 } {
  endpoint,b,err:=prepareReplay(session{SourceFormat:tc.format,Body:[]byte(tc.body)},defaultMessagesURL)
  if err!=nil || endpoint!="http://127.0.0.1:8317"+tc.path || gjson.GetBytes(b,tc.cap).Int()!=1 || gjson.GetBytes(b,"stream").Bool(){t.Fatalf("%s: %s %s %v",tc.format,endpoint,b,err)}
  if tc.format=="claude" && gjson.GetBytes(b,"thinking.type").String()!="adaptive" {t.Fatal("thinking changed")}
 }
}

func TestRenewSelectedSessionSurvivesNewConversationAndRestart(t *testing.T){
 resetSessionsForTest()
 originalFile:=stateFile;defer func(){stateFile=originalFile;persistSettings=false}()
 stateFile=filepath.Join(t.TempDir(),"sessions.json");persistSettings=true
 cfg.MaxSessions=1
 a:=http.Header{"X-Claude-Code-Session-Id":[]string{"selected"}}
 body:=[]byte(`{"max_tokens":100,"messages":[{"role":"user","content":"first"}]}`)
 upsertSession("claude-opus","claude","claude",a,body)
 id:=sessionKey("claude-opus",a,body)
 setSessionEnabled(id,true)
 upsertSession("claude-opus","claude","claude",http.Header{"X-Claude-Code-Session-Id":[]string{"other"}},body)
 if len(listSessions())!=1 || listSessions()[0].ID!=id {t.Fatal("selected session evicted")}
 mu.Lock();sessions=map[string]*session{};mu.Unlock()
 loadSessions()
 s,ok:=cloneSessionByID(id)
 if !ok || !s.Enabled || string(s.Body)!=string(body){t.Fatal("selection or snapshot lost")}
 stat,err:=os.Stat(stateFile);if err!=nil || stat.Mode().Perm()!=0600{t.Fatal("state permissions")}
}

func TestRenewLastRequestReschedulesAndFailureRetries(t *testing.T){
 resetSessionsForTest()
 now:=time.Now()
 s:=&session{ID:"0123456789abcdef",Enabled:true,LastSeen:now.Add(-51*time.Minute),SavedAt:now,Body:[]byte(`{"input":"hi"}`)}
 sessions[s.ID]=s
 if len(dueSnapshots(now,50*time.Minute))!=1{t.Fatal("due session missing")}
 s.LastSeen=now
 if len(dueSnapshots(now,50*time.Minute))!=0{t.Fatal("new request did not reset timer")}
 s.LastSeen=now.Add(-51*time.Minute)
 if !claimPing(cloneSession(s),false)||claimPing(cloneSession(s),true){t.Fatal("duplicate ping allowed")}
 recordSessionPing(s.ID,now,os.ErrDeadlineExceeded)
 if !s.LastPingAt.IsZero()||len(dueSnapshots(now.Add(30*time.Second),50*time.Minute))!=0||len(dueSnapshots(now.Add(61*time.Second),50*time.Minute))!=1 {t.Fatal("failed ping marked successful or retry incorrect")}
}

func TestRenewReadsActualResponseUsage(t *testing.T){
 resetSessionsForTest();old:=pingDo;defer func(){pingDo=old}()
 s:=session{ID:"0123456789abcdef",SavedAt:time.Now()};sessions[s.ID]=&s
 pingDo=func(_ string,_ http.Header,_ []byte)(int,[]byte,error){return 200,[]byte(`{"usage":{"input_tokens":10,"input_tokens_details":{"cached_tokens":350000}}}`),nil}
 if err:=replayForSession(s,defaultMessagesURL,[]byte(`{}`));err!=nil || s.LastPingCacheRead!=350000{t.Fatalf("cache usage missing: %v %+v",err,s)}
 pingDo=func(_ string,_ http.Header,_ []byte)(int,[]byte,error){return 200,[]byte(`{"status":"failed","error":{"message":"failure"}}`),nil}
 if replayForSession(s,defaultMessagesURL,nil)==nil {t.Fatal("failed response accepted")}
}

func TestRenewAutomaticCycleAtFiftyMinutes(t *testing.T){
 resetSessionsForTest();old:=pingDo;defer func(){pingDo=old}()
 now:=time.Now()
 s:=&session{ID:"0123456789abcdef",Enabled:true,Model:"claude-opus",SourceFormat:"openai-response",SavedAt:now,LastSeen:now.Add(-50*time.Minute-time.Second),Body:[]byte(`{"model":"claude-opus","input":"Hi"}`)}
 sessions[s.ID]=s
 calls:=0
 pingDo=func(endpoint string,h http.Header,b []byte)(int,[]byte,error){
  calls++
  if endpoint!="http://127.0.0.1:8317/v1/responses" || gjson.GetBytes(b,"max_output_tokens").Int()!=1 {t.Fatal("wrong replay")}
  return 200,[]byte(`{"usage":{"input_tokens":4000,"input_tokens_details":{"cached_tokens":3999}}}`),nil
 }
 sent,_,err:=pingOnce()
 if !sent || err!=nil || calls!=1 || s.LastPingAt.IsZero() || s.LastPingCacheRead!=3999 {t.Fatalf("automatic replay failed: %v",err)}
 pingOnce()
 if calls!=1 {t.Fatal("timer did not advance after success")}
 s.LastSeen=time.Now();s.LastPingAt=time.Time{}
 pingOnce()
 if calls!=1 {t.Fatal("new user request did not defer replay")}
 s.LastSeen=time.Now().Add(-51*time.Minute);s.Enabled=false
 pingOnce()
 if calls!=1 {t.Fatal("unselected conversation replayed")}
}

func TestRenewUsageFromConcurrentChatCannotChangeSessionResult(t *testing.T){
 resetSessionsForTest()
 s:=&session{ID:"0123456789abcdef",SavedAt:time.Now(),LastPingCacheRead:12345}
 sessions[s.ID]=s
 currentPingID=s.ID;keepaliveActive=1
 // Usage events lack session correlation; only the matching HTTP response
 // is allowed to update the row's cache verdict.
 handleUsage([]byte(`{"Provider":"claude","Model":"claude-opus","Detail":{"CacheCreationTokens":900000}}`))
 if s.LastPingCacheRead!=12345 || s.LastPingCacheWrite!=0 {t.Fatal("unrelated usage attributed to keepalive")}
}
