package main

import (
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

// Export the real plugin template with synthetic, credential-free sessions.
// This test is opt-in and never calls a model or installs a plugin.
func TestExportUIPreview(t *testing.T) {
 dir:=os.Getenv("CACHE_RENEW_PREVIEW_DIR")
 if dir=="" {t.Skip("preview export only")}
 resetSessionsForTest();defer resetSessionsForTest()
 cfg.MaxSessions=16
 guard:=false;cfg.GuardChat=&guard
 now:=time.Now()
 examples:=[]struct{id,label,model string; enabled bool;age time.Duration}{
  {"0123456789abcdef","节点搭建 · 缓存续期开发","claude-opus-5-5",true,30*time.Minute},
  {"1123456789abcdef","课程作业 · 数据分析","claude-sonnet-5-5",true,12*time.Minute},
  {"2123456789abcdef","文档整理","claude-haiku-5-5",false,5*time.Minute},
 }
 for _,x:=range examples{
  b:=claudeBody(x.label,"Preview only")
  if x.id=="1123456789abcdef" {b=[]byte(strings.ReplaceAll(string(b),`"ttl":"1h"`,`"ttl":"5m"`))}
  if x.id=="2123456789abcdef" {b=[]byte(`{"model":"claude-haiku-5-5","max_tokens":100,"messages":[{"role":"user","content":"文档整理"}]}`)}
  sessions[x.id]=&session{ID:x.id,Label:x.label,Model:x.model,Enabled:x.enabled,SourceFormat:"claude",Body:b,LastSeen:now.Add(-x.age),SavedAt:now.Add(-x.age),Info:inspectBody(b)}
 }
 for _,name:=range []string{"before.html","after.html"}{
  if name=="after.html"{sessions[examples[0].id].LastSeen=now;sessions[examples[0].id].SavedAt=now}
  html:=renderStatusHTML()
  // Preserve the embedded page; disable HTTP operations in this offline export.
  html=strings.Replace(html,"async function refresh(params){","async function refresh(params){return; /* offline screenshot fixture */",1)
  if err:=os.WriteFile(filepath.Join(dir,name),[]byte(html),0644);err!=nil{t.Fatal(err)}
 }
}
