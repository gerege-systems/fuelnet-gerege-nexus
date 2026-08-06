"use client";

import {useCallback,useEffect,useRef,useState} from "react";
import {QRCodeSVG} from "qrcode.react";
import {api} from "@/lib/api";
import {useI18n} from "@/lib/i18n";
import {Fingerprint,RefreshCw,ShieldCheck,Smartphone} from "lucide-react";

type Method="id"|"qr";
type Phase="idle"|"starting"|"waiting"|"expired"|"refused"|"error"|"success";
type Start={session_id:string;device_link_url?:string;verification_code:string;expires_at:string};

function mobile(){return typeof navigator!=="undefined"&&/Android|iPhone|iPad|iPod/i.test(navigator.userAgent)}
function callbackURL(){return typeof window==="undefined"?"":`${window.location.origin}/auth/eid/callback`}

export default function EIDLogin({next="/apps",compact=false}:{next?:string;compact?:boolean}){
  const {t}=useI18n();
  const [method,setMethod]=useState<Method>("id"),[phase,setPhase]=useState<Phase>("idle"),[nationalId,setNationalId]=useState(""),[start,setStart]=useState<Start|null>(null),[error,setError]=useState("");
  const timer=useRef<ReturnType<typeof setInterval>|null>(null),mounted=useRef(true);
  const stop=useCallback(()=>{if(timer.current){clearInterval(timer.current);timer.current=null}},[]);
  const poll=useCallback(async(sid:string)=>{try{const res=await api.pollEID(sid);if(!mounted.current)return;if(res.state==="COMPLETE"){stop();setPhase("success");if(res.token)localStorage.setItem("session_token",res.token);window.location.assign(next)}else if(res.state==="EXPIRED"){stop();setPhase("expired")}else if(res.state==="REFUSED"){stop();setPhase("refused")}}catch(e:any){if(!mounted.current)return;stop();setError(e.message||t("auth.message.error_link"));setPhase("error")}},[next,stop]);
  const arm=useCallback((data:Start)=>{setStart(data);setPhase("waiting");timer.current=setInterval(()=>void poll(data.session_id),2500);void poll(data.session_id)},[poll]);
  const begin=useCallback(async(selected:Method)=>{stop();setError("");setStart(null);setPhase("starting");try{const cb=mobile()?callbackURL():"";const data=selected==="qr"?await api.startEID(cb):await api.startEIDByNationalID(nationalId.trim().toUpperCase(),cb);arm(data);if(selected==="qr"&&mobile())window.location.href=`geregesmartid://approve?sessionId=${encodeURIComponent(data.session_id)}`}catch(e:any){setError(e.message||t("auth.message.error_service"));setPhase("error")}},[arm,nationalId,stop]);
  useEffect(()=>()=>{mounted.current=false;stop()},[stop]);
  const switchMethod=(value:Method)=>{stop();setMethod(value);setPhase("idle");setStart(null);setError("");if(value==="qr")void begin("qr")};
  const terminal=phase==="error"||phase==="expired"||phase==="refused";
  return <div className={`eid-login ${compact?"eid-login--compact":""}`} aria-live="polite">
    <header className="eid-login__head"><span><Fingerprint/></span><div><h2>{t("auth.eid.title")}</h2><p>{t("auth.eid.subtitle")}</p></div></header>
    <div className="eid-tabs" role="tablist"><button className={method==="id"?"is-active":""} onClick={()=>switchMethod("id")}>{t("auth.eid.tab_id")}</button><button className={method==="qr"?"is-active":""} onClick={()=>switchMethod("qr")}>{t("auth.eid.tab_qr")}</button></div>
    {error&&<p className="eid-alert eid-alert--error">{error}</p>}{phase==="expired"&&<p className="eid-alert">{t("auth.message.expired")}</p>}{phase==="refused"&&<p className="eid-alert eid-alert--error">{t("auth.message.refused")}</p>}
    {method==="id"&&phase!=="waiting"&&phase!=="starting"&&<form onSubmit={e=>{e.preventDefault();void begin("id")}}><label htmlFor="eid-rd">{t("auth.eid.reg_number")}</label><input id="eid-rd" value={nationalId} onChange={e=>setNationalId(e.target.value.toUpperCase())} placeholder={t("auth.eid.reg_number_placeholder")} autoComplete="off" required minLength={8}/><button className="eid-primary"><Smartphone/> {t("auth.eid.send_request")}</button></form>}
    {phase==="starting"&&<div className="eid-status"><RefreshCw className="animate-spin"/> {t("auth.message.starting")}</div>}
    {phase==="waiting"&&start&&<div className="eid-wait">
      {method==="qr"&&start.device_link_url&&<div className="eid-qr"><QRCodeSVG value={start.device_link_url} size={compact?154:190} level="M"/></div>}
      <p>{t(method==="qr"?"auth.message.scan_qr":"auth.message.sent_push")}</p><small>{t("auth.eid.verification_code")}</small><strong>{start.verification_code}</strong><span><ShieldCheck/> {t("auth.eid.confirm_hint")}</span>
    </div>}
    {phase==="success"&&<p className="eid-alert eid-alert--success">{t("auth.message.success")}</p>}
    {terminal&&<button className="eid-retry" onClick={()=>void begin(method)}><RefreshCw/> {t("auth.action.retry")}</button>}
    <footer><ShieldCheck/> {t("auth.eid.footer")}</footer>
  </div>
}
