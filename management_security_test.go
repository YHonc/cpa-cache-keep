package main

import (
 "bytes"
 "encoding/json"
 "net/http"
 "net/url"
 "strings"
 "testing"
 "github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func managementTestCall(t *testing.T,req pluginapi.ManagementRequest)pluginapi.ManagementResponse{
 t.Helper();raw,_:=json.Marshal(req);out,err:=handleStatusRequest(raw);if err!=nil{t.Fatal(err)}
 var env envelope;json.Unmarshal(out,&env);var resp pluginapi.ManagementResponse
 if err:=json.Unmarshal(env.Result,&resp);err!=nil{t.Fatal(err)};return resp
}

func TestResourceIsStaticAndIgnoresAllLegacyActions(t *testing.T){
 resetSessionsForTest();defer resetSessionsForTest()
 id:="0123456789abcdef";sessions[id]=&session{ID:id,Label:"PRIVATE-SENTINEL",Enabled:true,Body:[]byte(`{"input":"PRIVATE-BODY"}`)}
 baseline:=managementTestCall(t,pluginapi.ManagementRequest{Method:"GET",Path:resourceStatusPath})
 for _,query:=range []url.Values{{"format":{"json"}},{"toggle":{id},"on":{"0"}},{"forget":{id}},{"rename":{id},"name":{"bad"}},{"ping":{id}},{"stop":{"70"}},{"guard":{"1"}}}{
  resp:=managementTestCall(t,pluginapi.ManagementRequest{Method:"GET",Path:resourceStatusPath,Query:query})
  if !bytes.Equal(baseline.Body,resp.Body)||resp.StatusCode!=200{t.Fatal("resource varies with state/action query")}
  if len(sessions)!=1||!sessions[id].Enabled||sessions[id].Label!="PRIVATE-SENTINEL"{t.Fatal("resource mutated state")}
 }
 if strings.Contains(string(baseline.Body),"PRIVATE-"){t.Fatal("resource leaks private state")}
 resp:=managementTestCall(t,pluginapi.ManagementRequest{Method:"POST",Path:resourceStatusPath,Body:[]byte(`{"action":"forget","id":"`+id+`"}`)})
 if resp.StatusCode!=405||len(sessions)!=1{t.Fatal("resource accepted POST")}
}

func TestManagementGetCannotMutate(t *testing.T){
 resetSessionsForTest();defer resetSessionsForTest()
 id:="0123456789abcdef";sessions[id]=&session{ID:id}
 resp:=managementTestCall(t,pluginapi.ManagementRequest{Method:"GET",Path:managementActionsPath,Query:url.Values{"toggle":{id},"on":{"1"}}})
 if resp.StatusCode!=405||sessions[id].Enabled{t.Fatal("GET action accepted")}
 resp=managementTestCall(t,pluginapi.ManagementRequest{Method:"GET",Path:managementStatePath,Query:url.Values{"forget":{id}}})
 if resp.StatusCode!=http.StatusOK||len(sessions)!=1{t.Fatal("GET state mutated session")}
}
