"use client";
import {useEffect,useState} from "react";
import {useRouter} from "next/navigation";
import Link from "next/link";
import EIDLogin from "@/components/EIDLogin";
import LanguageSwitcher from "@/components/LanguageSwitcher";
import {api} from "@/lib/api";
import {resetAccess} from "@/lib/access";
import {useI18n} from "@/lib/i18n";
import type {TranslationKey} from "@/lib/i18n";
import {ChevronDown,Lock,Mail,ShieldCheck} from "lucide-react";

/** Серверийн богино кодыг хүн уншихаар мессеж болгоно. Танихгүй кодыг — жишээ
    нь провайдерийн өөрийнх нь илгээсэн ер бусын алдааг — ерөнхий мэдэгдэл
    авна: код нь бүртгэлд үлддэг, дэлгэц дээр гарах ёсгүй. */
const SSO_ERRORS:Record<string,TranslationKey>={no_account:"auth.sso.error_no_account",provider_unreachable:"auth.sso.error_unreachable",stale_request:"auth.sso.error_stale",access_denied:"auth.sso.error_denied"};

export default function LoginPage(){const router=useRouter();const {t}=useI18n();const [next,setNext]=useState("/apps"),[admin,setAdmin]=useState(false),[email,setEmail]=useState("admin@example.com"),[password,setPassword]=useState("Password123!"),[error,setError]=useState("");
  // undefined = хараахан асуугаагүй. Энэ ялгаа чухал: асуухаас өмнө eID
  // хэлбэрийг зурчихвал холбоосон суулгац дээр хүн энд нэвтэрч болно гэж
  // хэсэг хугацаанд итгэж, дараа нь өөр рүү шилжсэн нь будлиантай.
  const [sso,setSSO]=useState<{enabled:boolean;provider_name?:string;start_url?:string;local_login:boolean}|undefined>();
  useEffect(()=>{const requested=new URLSearchParams(location.search).get("next");if(requested?.startsWith("/")&&!requested.startsWith("//"))setNext(requested);
    const failed=new URLSearchParams(location.search).get("sso_error");if(failed)setError(t(SSO_ERRORS[failed]||"auth.sso.error_generic"));
    // Алдаагаа өөрөө барина: тохиргоо ирэхгүй бол энэ суулгац өөрөө нэвтрүүлдэг
    // гэж үзнэ — эс бөгөөс API-гийн түр саатал нэвтрэх дэлгэцийг хоослоно.
    void api.ssoConfig().then(setSSO).catch(()=>setSSO({enabled:false,local_login:true}))},[t]);
  // Провайдер руу шилжих нь энэ дэлгэцийн ажил дуусгах цэг: цаашид энд юу ч
  // асуухгүй тул нэмэлт товч харуулахгүйгээр шууд илгээнэ.
  useEffect(()=>{if(sso?.enabled&&!sso.local_login&&sso.start_url&&!error)window.location.assign(`${sso.start_url}?next=${encodeURIComponent(next)}`)},[sso,next,error]);
  function startSSO(){if(sso?.start_url)window.location.assign(`${sso.start_url}?next=${encodeURIComponent(next)}`)}
  // resetAccess before the push: router.push is a client-side navigation, so
  // whatever the previous session left cached would answer for this one.
  async function passwordLogin(e:React.FormEvent){e.preventDefault();setError("");try{await api.login(email,password);resetAccess();
    // /oauth2/* belongs to the API, not to this Next app, so a client-side
    // push would 404 instead of resuming the authorization request. eID sign-in
    // already navigates for real; this path has to as well.
    if(next.startsWith("/oauth2/"))window.location.assign(next);else router.push(next)}catch(err:any){setError(err.message||t("auth.message.error_password"))}}

  const federated=sso?.enabled===true;
  const provider=sso?.provider_name||"";
  // Холбоосон, орон нутгийн нэвтрэлтгүй суулгац дээр энэ дэлгэц бол зөвхөн
  // дамжуулах цэг. Алдаа гарсан үед л энд үлдэж, юу болсныг хэлнэ.
  const redirecting=federated&&!sso?.local_login&&!error;
  return <main className="eid-page"><div className="eid-page__pattern"/><header><Link href="/" className="gp-brand"><img src="/brand.webp" alt=""/><span>Gerege Nexus</span></Link><LanguageSwitcher variant="dark"/></header><section className="eid-page__content"><div className="eid-page__intro"><span className="gp-eyebrow"><i/> {t("auth.view.eyebrow")}</span><h1>{t("auth.view.title_lead")}<br/><em>{t("auth.view.title_highlight")}</em> {t("auth.view.title_tail")}</h1><p>{federated?t("auth.sso.lede",{provider}):t("auth.view.lede")}</p><ul><li>{t("auth.view.point_push")}</li><li>{t("auth.view.point_qr")}</li><li>{t("auth.view.point_rbac")}</li></ul></div><div>
    {sso===undefined&&<p className="admin-login__pending">{t("auth.sso.checking")}</p>}
    {redirecting&&<p className="admin-login__pending">{t("auth.sso.redirecting",{provider})}</p>}
    {federated&&!redirecting&&<div className="sso-card"><ShieldCheck/><strong>{t("auth.sso.card_title",{provider})}</strong><p>{t("auth.sso.card_body",{provider})}</p>{error&&<p className="sso-card__error">{error}</p>}<button onClick={startSSO}>{t("auth.sso.sign_in",{provider})}</button></div>}
    {sso&&(!federated||sso.local_login)&&<EIDLogin next={next}/>}
    {/* Холбоосон суулгац дээр орон нутгийн хэлбэрүүд зөвхөн операторын
        SSO_CLIENT_LOCAL_LOGIN нээсэн үед л гарна — провайдер унасан үеийн
        буцах зам. */}
    {sso&&(!federated||sso.local_login)&&<><button className="admin-disclosure" onClick={()=>setAdmin(v=>!v)}><Lock/> {t("auth.action.admin_disclosure")} <ChevronDown className={admin?"rotate-180":""}/></button>{admin&&<form className="admin-login" onSubmit={passwordLogin}>{error&&<p>{error}</p>}<label><Mail/> <input type="email" value={email} onChange={e=>setEmail(e.target.value)} required/></label><label><Lock/> <input type="password" value={password} onChange={e=>setPassword(e.target.value)} required/></label><button>{t("auth.action.admin_sign_in")}</button></form>}</>}
  </div></section></main>}
