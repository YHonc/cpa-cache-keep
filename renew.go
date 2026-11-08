package main

// Local adaptation of Banben07/cpa_claude_cache_keep (MIT).
// Capture occurs before protocol translation in CPA 8.0.59. Replay the same
// client protocol and public alias through the gateway, never the upstream URL.
import (
 "encoding/json"
 "fmt"
 "net/url"
 "os"
 "path/filepath"
 "strings"
 "sync"
 "time"

 "github.com/tidwall/gjson"
 "github.com/tidwall/sjson"
)

var stateFile = "claude-cache-keepalive.sessions.json"
var diskMu sync.Mutex

func setReplayModel(body []byte, model string) ([]byte, error) {
 return sjson.SetBytes(body, "model", model)
}

func prepareReplay(s session, base string) (string, []byte, error) {
 u, err := url.Parse(base)
 if err != nil || u.Host == "" { return "", nil, fmt.Errorf("保活地址无效") }
 // No redirects or non-loopback targets: snapshots contain client credentials.
 if u.Scheme != "http" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" && u.Hostname() != "localhost") {
  return "", nil, fmt.Errorf("保活地址必须为本机 HTTP loopback")
 }
 b := append([]byte(nil), s.Body...)
 format := strings.ToLower(s.SourceFormat)
 cap := "max_tokens"
 switch {
 case strings.Contains(format, "response") || gjson.GetBytes(b,"input").Exists():
  u.Path = "/v1/responses"
  cap = "max_output_tokens"
 case format == "openai" || strings.Contains(format,"chat-completion"):
  u.Path = "/v1/chat/completions"
  if gjson.GetBytes(b,"max_completion_tokens").Exists() { cap = "max_completion_tokens" }
 case format == "claude" || format == "anthropic" || format == "":
  u.Path = "/v1/messages"
 default:
  return "",nil,fmt.Errorf("暂不支持重放格式 %s",s.SourceFormat)
 }
 b,err = sjson.SetBytes(b,cap,1)
 if err != nil { return "",nil,err }
 b,err = sjson.SetBytes(b,"stream",false)
 if cap == "max_output_tokens" { b,_ = sjson.SetBytes(b,"background",false) }
 u.RawQuery = ""; u.Fragment = ""
 return u.String(),b,err
}

func claimPing(s session, manual bool) bool {
 mu.Lock(); defer mu.Unlock()
 item := sessions[s.ID]
 if item == nil || item.InFlight || !item.SavedAt.Equal(s.SavedAt) { return false }
 if !manual && (!item.Enabled || item.PingExpensive || time.Now().Before(item.RetryAt)) { return false }
 item.InFlight = true
 item.LastAttemptAt = time.Now()
 return true
}

func replayForSession(s session, endpoint string, body []byte) error {
 status, raw, err := pingDo(endpoint,copyPingHeaders(s.Headers),body)
 if err != nil { return err }
 if status < 200 || status >= 300 {
  // Upstream error bodies can echo request contents; retain the error type only.
  typ := gjson.GetBytes(raw,"error.type").String()
  return fmt.Errorf("保活 HTTP %d %s",status,typ)
 }
 if !gjson.ValidBytes(raw) || gjson.GetBytes(raw,"error").Exists() || gjson.GetBytes(raw,"status").String()=="failed" {
  return fmt.Errorf("保活响应不是有效补全")
 }
 usage:=gjson.GetBytes(raw,"usage")
 if !usage.IsObject() { return fmt.Errorf("响应缺少 usage，无法确认保活；稍后重试") }
 read:=usage.Get("cache_read_input_tokens").Int()
 write:=usage.Get("cache_creation_input_tokens").Int()
 if cached:=usage.Get("input_tokens_details.cached_tokens");cached.Exists(){read=cached.Int()}
 if cached:=usage.Get("prompt_tokens_details.cached_tokens");cached.Exists(){read=cached.Int()}
 mu.Lock(); defer mu.Unlock()
 item:=sessions[s.ID]
 if item==nil || !item.SavedAt.Equal(s.SavedAt) { return nil }
 applyKeepaliveCacheResult(item,read,write)
	item.WriteTTLKnown = usage.Get("cache_creation.ephemeral_5m_input_tokens").Exists() || usage.Get("cache_creation.ephemeral_1h_input_tokens").Exists()
	item.LastWrite5m = usage.Get("cache_creation.ephemeral_5m_input_tokens").Int()
	item.LastWrite1h = usage.Get("cache_creation.ephemeral_1h_input_tokens").Int()
 switch {
 case read>0: item.Result="缓存读取已确认"
 case write>0: item.Result="发生缓存写入，未确认读取旧缓存"
 default: item.Result="请求成功，未确认缓存命中"
 }
 return nil
}

// diskMu serializes writers before taking a snapshot, so an older snapshot
// cannot overwrite a more recent selection. State includes replay bodies and
// client authorization: keep on the CPA host with permissions 0600.
func persistSessionsLocked() {
 if !persistSettings || stateFile=="" { return }
 diskMu.Lock(); defer diskMu.Unlock()
 mu.Lock()
 list:=make([]session,0,len(sessions))
 for _,s:=range sessions { if s!=nil { list=append(list,cloneSession(s)) } }
 mu.Unlock()
 raw,err:=json.Marshal(list)
 if err==nil {
  tmp:=stateFile+".tmp"
  err=os.WriteFile(tmp,raw,0600)
  if err==nil { err=os.Chmod(tmp,0600) }
  if err==nil { err=os.Rename(tmp,stateFile) }
 }
 if err!=nil { mu.Lock();lastErr="会话持久化失败："+err.Error();mu.Unlock() }
}

func loadSessions() {
 if !persistSettings || stateFile=="" { return }
 diskMu.Lock();defer diskMu.Unlock()
 raw,err:=os.ReadFile(filepath.Clean(stateFile))
 if os.IsNotExist(err) { return }
 var list []session
 if err==nil { err=json.Unmarshal(raw,&list) }
 mu.Lock(); defer mu.Unlock()
 if err!=nil { lastErr="读取会话失败："+err.Error();return }
 for i:=range list {
  s:=list[i]
  if sanitizeSessionID(s.ID)=="" || len(s.Body)==0 { continue }
  s.InFlight=false
  if _,exists:=sessions[s.ID];!exists { sessions[s.ID]=&s }
 }
}
