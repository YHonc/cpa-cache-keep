package main

import (
 "encoding/json"
 "net/http"
 "strings"
 "testing"
 "time"

 "github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestLiveUIUsesRealStateWithoutReplaySecrets(t *testing.T){
 resetSessionsForTest();defer resetSessionsForTest()
 body:=[]byte(`{"max_tokens":100,"messages":[{"role":"user","content":"Synthetic conversation"}]}`)
 upsertSession("claude-opus","claude","claude",http.Header{"X-Claude-Code-Session-Id":[]string{"synthetic-id"},"Authorization":[]string{"Bearer PRIVATE-TEST-KEY"}},body)
 raw,_:=json.Marshal(publicStatus())
 if strings.Contains(string(raw),"PRIVATE-TEST-KEY")||strings.Contains(string(raw),"Headers")||strings.Contains(string(raw),"Body") {t.Fatal("public state leaked replay data")}
 html:=renderStatusHTML()
 for _,want:=range []string{"切换明暗主题","会话识别与缓存详情"}{if !strings.Contains(html,want){t.Fatalf("UI missing %s",want)}}
 if strings.Contains(html,"Synthetic conversation")||strings.Contains(html,"PRIVATE-TEST-KEY"){t.Fatal("static resource leaks session state")}
 if strings.Contains(html,"模拟新请求")||strings.Contains(html,"交互预览") {t.Fatal("demo controls shipped to live UI")}
}

func TestLiveManagementJSONToggleAndCountdown(t *testing.T){
 resetSessionsForTest();defer resetSessionsForTest()
 now:=time.Now()
 id:="0123456789abcdef"
 sessions[id]=&session{ID:id,Label:"test",Body:[]byte(`{"input":"test"}`),LastSeen:now,SavedAt:now}
 req,_:=json.Marshal(pluginapi.ManagementRequest{Method:"POST",Path:managementActionsPath,Body:[]byte(`{"action":"toggle","id":"`+id+`","enabled":true}`)})
 raw,err:=handleStatusRequest(req);if err!=nil{t.Fatal(err)}
 var env envelope;json.Unmarshal(raw,&env)
 var resp pluginapi.ManagementResponse;json.Unmarshal(env.Result,&resp)
 var state publicState;if err:=json.Unmarshal(resp.Body,&state);err!=nil{t.Fatal(err)}
 if resp.StatusCode!=200||len(state.Items)!=1||!state.Items[0].Active {t.Fatal("toggle did not persist in JSON response")}
 if delta:=state.Items[0].Next-now.UnixMilli();delta!=50*60*1000{t.Fatalf("wrong next request: %d",delta)}
}
