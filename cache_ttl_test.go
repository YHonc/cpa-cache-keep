package main

import (
 "html"
 "net/http"
 "strings"
 "testing"
 "time"
)

func TestDeclaredCacheTTL(t *testing.T){
 for _,tc:=range []struct{body,want string}{
  {`{"system":[{"cache_control":{"type":"ephemeral","ttl":"1h"}}]}`,"1h"},
  {`{"cache_control":{"type":"ephemeral"}}`,"5m"},
  {`{"tools":[{"cache_control":{"type":"ephemeral","ttl":"5m"}}]}`,"5m"},
  {`{"system":[{"cache_control":{"type":"ephemeral","ttl":"1h"}}],"messages":[{"content":[{"cache_control":{"type":"ephemeral"}}]}]}`,"mixed"},
  {`{"messages":[{"content":"example: cache_control ttl 1h"}]}`,""},
  {`{"cache_control":{"type":"ephemeral","ttl":"5h"}}`,"unknown"},
 }{if got:=declaredCacheTTL([]byte(tc.body));got!=tc.want{t.Errorf("got %q want %q",got,tc.want)}}
}

func TestCacheTTLResponseAndUIEvidence(t *testing.T){
 resetSessionsForTest();defer resetSessionsForTest()
 old:=pingDo;defer func(){pingDo=old}()
 s:=session{ID:"0123456789abcdef",SavedAt:time.Now(),Info:bodyInfo{CacheTTL:"mixed"}}
 sessions[s.ID]=&s
 pingDo=func(_ string,_ http.Header,_ []byte)(int,[]byte,error){return 200,[]byte(`{"usage":{"cache_read_input_tokens":2000,"cache_creation_input_tokens":100,"cache_creation":{"ephemeral_1h_input_tokens":80,"ephemeral_5m_input_tokens":20}}}`),nil}
 if err:=replayForSession(s,defaultMessagesURL,nil);err!=nil{t.Fatal(err)}
 if !s.WriteTTLKnown || s.LastWrite1h!=80 || s.LastWrite5m!=20 {t.Fatal("write TTL breakdown missing")}
 rendered:=html.UnescapeString(renderLegacyStatusHTML())
 for _,want:=range []string{"混合 · 1h + 5m","50 分钟续期只能覆盖 1h 部分","上游未提供时长拆分","1h 80 token / 5m 20 token"}{
  if !strings.Contains(rendered,want){t.Fatalf("missing UI evidence %s",want)}
 }
}
