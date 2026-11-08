package main

import (
 _ "embed"
 "encoding/json"
 "fmt"
 "strings"
 "time"
)

//go:embed ui.html
var liveHTML string

type publicSession struct {
 ID string `json:"id"`
 Name string `json:"name"`
 Model string `json:"model"`
 TTL string `json:"ttl"`
 Active bool `json:"active"`
 Last int64 `json:"last"`
 Next int64 `json:"next"`
 Paused bool `json:"paused"`
 Error string `json:"error"`
 Detail string `json:"detail"`
}

type publicState struct {
 Version string `json:"version"`
 Now int64 `json:"now"`
 Capacity int `json:"capacity"`
 Interval int `json:"interval"`
 Paused bool `json:"paused"`
 PauseReason string `json:"pauseReason"`
 ActionError string `json:"actionError,omitempty"`
 Items []publicSession `json:"items"`
}

// The browser never receives replay bodies, headers, or client API keys.
func publicStatus() publicState {
 st:=currentStatus()
 result:=publicState{Version:pluginVersion,Now:st.Now.UnixMilli(),Capacity:st.MaxSessions,Interval:st.IntervalMin,Paused:st.Budget.KeepalivePaused,PauseReason:st.Budget.PauseReason,Items:[]publicSession{}}
 for _,s:=range st.Sessions {
  ttl:=s.Info.CacheTTL;if ttl==""{ttl="unknown"}
  next:=sessionNextPingAt(s.LastSeen,s.LastPingAt,st.Now,time.Duration(st.IntervalMin)*time.Minute)
  if s.RetryAt.After(next){next=s.RetryAt}
  identity:="内容指纹识别：缺少会话 ID，压缩上下文或更换模型可能改变归组。"
  if claudeSessionID(s.Headers,s.Body)!=""{identity="按客户端会话 ID 识别。"}
  detail:=fmt.Sprintf("%s %s 最近保活：%s。%s 读取缓存的实际 TTL：上游未提供时长拆分。",identity,cacheTTLHint(ttl),formatTime(s.LastPingAt),cacheVerdict(s.LastPingCacheRead,s.LastPingCacheWrite))
  if s.WriteTTLKnown{detail+=fmt.Sprintf(" 最近保活写入：1h %s token / 5m %s token。",formatUnits(s.LastWrite1h),formatUnits(s.LastWrite5m))}
  if s.LastPingError!=""{detail+=" 最近错误："+s.LastPingError}
  if s.PingExpensive{detail+=" 连续两次大量写缓存，自动续期已暂停；新请求或手动续期可以恢复。"}
  result.Items=append(result.Items,publicSession{ID:s.ID,Name:s.Label,Model:s.Model,TTL:ttl,Active:s.Enabled,Last:s.LastSeen.UnixMilli(),Next:next.UnixMilli(),Paused:s.PingExpensive,Error:s.LastPingError,Detail:detail})
 }
 return result
}

func renderLiveHTML() string {
 raw,_:=json.Marshal(publicStatus())
 return strings.Replace(liveHTML,"/*CACHE_STATE*/null",string(raw),1)
}

func renderStatusHTML() string { return renderLiveHTML() }
