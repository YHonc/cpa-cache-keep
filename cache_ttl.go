package main

import "github.com/tidwall/gjson"

// Inspect structured cache_control blocks only. A quoted example inside a
// user's message is not evidence of the request's caching policy.
func declaredCacheTTL(body []byte) string {
 short,long,unknown:=false,false,false
 var visit func(gjson.Result)
 visit=func(v gjson.Result){
  if !v.IsObject() && !v.IsArray(){return}
  v.ForEach(func(key,child gjson.Result)bool{
   if key.String()=="cache_control" && child.IsObject(){
    if child.Get("type").String()!="ephemeral"{unknown=true;return true}
    switch child.Get("ttl").String(){
    case "","5m":short=true
    case "1h":long=true
    default:unknown=true
    }
   }else{visit(child)}
   return true
  })
 }
 visit(gjson.ParseBytes(body))
 if unknown{return "unknown"}
 if short&&long{return "mixed"}
 if long{return "1h"}
 if short{return "5m"}
 return ""
}

func cacheTTLLabel(ttl string)string{
 switch ttl{
 case "1h":return "1h · 1 小时"
 case "5m":return "5m · 5 分钟"
 case "mixed":return "混合 · 1h + 5m"
 default:return "未知 · 请求未声明或无法识别"
 }
}

func cacheTTLHint(ttl string)string{
 switch ttl{
 case "5m":return "当前请求声明 5 分钟缓存，50 分钟续期无法覆盖；需先让客户端或 CPA 使用 1h。"
 case "mixed":return "请求包含 1h 和 5m 缓存；50 分钟续期只能覆盖 1h 部分。"
 case "1h":return "请求声明为 1 小时；上游最终 TTL 可能受 CPA 转换影响。"
 default:return "此快照没有可确认的 TTL；不能仅凭缓存读取量推断为 1h。"
 }
}
