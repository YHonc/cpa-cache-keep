package main

import (
 "encoding/json"
 "net/http"
 "strings"

 "github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const resourceStatusPath="/v0/resource/plugins/"+pluginID+"/status"
const managementStatePath="/v0/management/plugins/"+pluginID+"/state"
const managementActionsPath="/v0/management/plugins/"+pluginID+"/actions"

type managementAction struct {
 Action string `json:"action"`
 ID string `json:"id"`
 Enabled *bool `json:"enabled,omitempty"`
 Name string `json:"name,omitempty"`
 StopPercent *int `json:"stop_percent,omitempty"`
 GuardChat *bool `json:"guard_chat,omitempty"`
}

func jsonManagementResponse(status int,payload any)([]byte,error){
 body,err:=json.Marshal(payload);if err!=nil{return nil,err}
 return okEnvelope(pluginapi.ManagementResponse{StatusCode:status,Headers:http.Header{"Content-Type":[]string{"application/json; charset=utf-8"},"Cache-Control":[]string{"no-store"}},Body:body})
}

// CPA authenticates registered management routes before invoking the plugin.
// Resource handling is an independent branch: it never inspects state or actions.
func handleStatusRequest(raw []byte)([]byte,error){
 var req pluginapi.ManagementRequest
 if err:=json.Unmarshal(raw,&req);err!=nil{return jsonManagementResponse(400,map[string]string{"error":"invalid request"})}
 if req.Path==resourceStatusPath {
  if req.Method!=http.MethodGet{return jsonManagementResponse(405,map[string]string{"error":"method not allowed"})}
  return okEnvelope(pluginapi.ManagementResponse{StatusCode:200,Headers:http.Header{"Content-Type":[]string{"text/html; charset=utf-8"},"Cache-Control":[]string{"no-cache"}},Body:[]byte(liveHTML)})
 }
 if req.Path==managementStatePath&&req.Method==http.MethodGet{return jsonManagementResponse(200,publicStatus())}
 if req.Path!=managementActionsPath{return jsonManagementResponse(404,map[string]string{"error":"not found"})}
 if req.Method!=http.MethodPost{return jsonManagementResponse(405,map[string]string{"error":"method not allowed"})}
 var action managementAction
 decoder:=json.NewDecoder(strings.NewReader(string(req.Body)));decoder.DisallowUnknownFields()
 if err:=decoder.Decode(&action);err!=nil{return jsonManagementResponse(400,map[string]string{"error":"invalid action JSON"})}
 if action.Action=="settings" {
  if action.StopPercent==nil&&action.GuardChat==nil{return jsonManagementResponse(400,map[string]string{"error":"settings missing"})}
  if action.StopPercent!=nil {
   if *action.StopPercent<minStopPercent||*action.StopPercent>maxStopPercent{return jsonManagementResponse(400,map[string]string{"error":"invalid stop_percent"})}
   setStopPercent(*action.StopPercent)
  }
  if action.GuardChat!=nil{setGuardChat(*action.GuardChat)}
  return jsonManagementResponse(200,publicStatus())
 }
 id:=sanitizeSessionID(action.ID)
 if id==""{return jsonManagementResponse(400,map[string]string{"error":"invalid session ID"})}
 if _,ok:=cloneSessionByID(id);!ok{return jsonManagementResponse(404,map[string]string{"error":"session not found"})}
 switch action.Action {
 case "toggle":
  if action.Enabled==nil{return jsonManagementResponse(400,map[string]string{"error":"enabled missing"})}
  setSessionEnabled(id,*action.Enabled)
 case "forget":forgetSession(id)
 case "rename":renameSession(id,action.Name)
 case "ping":
  if err:=pingSessionNow(id);err!=nil{return jsonManagementResponse(502,map[string]string{"error":err.Error()})}
 default:return jsonManagementResponse(400,map[string]string{"error":"unknown action"})
 }
 return jsonManagementResponse(200,publicStatus())
}
