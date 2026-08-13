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
import {ChevronDown,HelpCircle,Lock,Mail,ShieldCheck} from "lucide-react";

/** Серверийн богино кодыг хүн уншихаар мессеж болгоно. Танихгүй кодыг — жишээ
    нь провайдерийн өөрийнх нь илгээсэн ер бусын алдааг — ерөнхий мэдэгдэл
    авна: код нь бүртгэлд үлддэг, дэлгэц дээр гарах ёсгүй. */
const SSO_ERRORS:Record<string,TranslationKey>={no_account:"auth.sso.error_no_account",provider_unreachable:"auth.sso.error_unreachable",stale_request:"auth.sso.error_stale",access_denied:"auth.sso.error_denied",email_unverified:"auth.sso.error_email_unverified",domain_not_allowed:"auth.sso.error_domain_not_allowed"};

/** Google-ийн албан ёсны дөрвөн өнгийн "G". */
function GoogleMark(){return <svg viewBox="0 0 48 48" aria-hidden="true"><path fill="#EA4335" d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"/><path fill="#4285F4" d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"/><path fill="#FBBC05" d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"/><path fill="#34A853" d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"/></svg>}

export default function LoginPage(){const router=useRouter();const {t}=useI18n();const [next,setNext]=useState("/profile"),[admin,setAdmin]=useState(false),[email,setEmail]=useState("admin@example.com"),[password,setPassword]=useState("Password123!"),[error,setError]=useState("");
  // undefined = хараахан асуугаагүй. Энэ ялгаа чухал: асуухаас өмнө eID
  // хэлбэрийг зурчихвал холбоосон суулгац дээр хүн энд нэвтэрч болно гэж
  // хэсэг хугацаанд итгэж, дараа нь өөр рүү шилжсэн нь будлиантай.
  const [sso,setSSO]=useState<{enabled:boolean;provider_name?:string;start_url?:string;local_login:boolean;google?:{enabled:boolean;start_url?:string}}|undefined>();
  // Хэн асууж байна. Зөвхөн authorization хүсэлтээс ирсэн үед л утгатай, ба
  // нэрийг нь серверээс асууна — `next` дотор ирсэн client_id-г л ашиглаж,
  // дэлгэц дээр гарах нэрийг хаяг тодорхойлохыг зөвшөөрөхгүй.
  const [asker,setAsker]=useState<{client_name:string}|null>(null);

  useEffect(()=>{const requested=new URLSearchParams(location.search).get("next");if(requested?.startsWith("/")&&!requested.startsWith("//"))setNext(requested);
    const failed=new URLSearchParams(location.search).get("sso_error");if(failed)setError(t(SSO_ERRORS[failed]||"auth.sso.error_generic"));
    // Алдаагаа өөрөө барина: тохиргоо ирэхгүй бол энэ суулгац өөрөө нэвтрүүлдэг
    // гэж үзнэ — эс бөгөөс API-гийн түр саатал нэвтрэх дэлгэцийг хоослоно.
    void api.ssoConfig().then(setSSO).catch(()=>setSSO({enabled:false,local_login:true,google:{enabled:false}}))},[t]);

  useEffect(()=>{if(!next.startsWith("/oauth2/auth"))return;
    const clientID=new URLSearchParams(next.slice(next.indexOf("?")+1)).get("client_id");
    if(!clientID)return;
    // Танихгүй client бол чимээгүй өнгөрнө: нэвтрэх дэлгэц ажилласаар байх ёстой
    // ба буруу client_id-г /oauth2/auth өөрөө татгалзана.
    void api.oauthClientInfo(clientID).then(setAsker).catch(()=>{})},[next]);

  // Провайдер руу шилжих нь энэ дэлгэцийн ажил дуусгах цэг: цаашид энд юу ч
  // асуухгүй тул нэмэлт товч харуулахгүйгээр шууд илгээнэ.
  useEffect(()=>{if(sso?.enabled&&!sso.local_login&&sso.start_url&&!error)window.location.assign(`${sso.start_url}?next=${encodeURIComponent(next)}`)},[sso,next,error]);
  function startSSO(){if(sso?.start_url)window.location.assign(`${sso.start_url}?next=${encodeURIComponent(next)}`)}
  function startGoogle(){if(sso?.google?.start_url)window.location.assign(`${sso.google.start_url}?next=${encodeURIComponent(next)}`)}
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
  const showLocal=!!sso&&(!federated||sso.local_login);

  return <main className="signin-shell">
    <header className="signin-shell__nav">
      <Link href="/" className="gp-brand"><img src="/brand.webp" alt=""/><span>Gerege Nexus</span></Link>
      <LanguageSwitcher/>
    </header>
    <section className="signin-shell__body">
      <div className="signin-card">
        {/* Хэн асууж байна. Authorization хүсэлтээс ирээгүй бол платформ өөрөө
            асууж байна гэсэн үг — тэр үед ч гэсэн карт нэгэн ижил харагдана. */}
        <div className="signin-card__asker">
          <strong>{asker?.client_name||t("auth.view.platform_name")}</strong>
          <span>{t("auth.signin.asker_note")}</span>
        </div>
        <hr className="signin-card__rule"/>

        {sso===undefined&&<p className="admin-login__pending">{t("auth.sso.checking")}</p>}

        {redirecting&&<>
          <div><h1 className="signin-card__title">{t("auth.signin.title")}</h1><p className="signin-card__lede">{t("auth.sso.redirecting",{provider})}</p></div>
        </>}

        {federated&&!redirecting&&<>
          <div><h1 className="signin-card__title">{t("auth.signin.title")}</h1><p className="signin-card__lede">{t("auth.sso.card_body",{provider})}</p></div>
          {error&&<p className="signin-alert">{error}</p>}
          <button className="signin-btn signin-btn--eid" onClick={startSSO}><ShieldCheck size={18}/> {t("auth.sso.sign_in",{provider})}</button>
        </>}

        {showLocal&&<>
          {!federated&&<div><h1 className="signin-card__title">{t("auth.signin.title")}</h1><p className="signin-card__lede">{t("auth.signin.lede")}</p></div>}
          {error&&!federated&&<p className="signin-alert">{error}</p>}
          <EIDLogin next={next} variant="signin"/>

          {/* Google. Сервер тохируулсан үед л гарна: тохируулаагүй байхад
              дарж болох мөртлөө юу ч болдоггүй товч харуулах нь амлалт биш,
              эвдрэл. */}
          {sso?.google?.enabled&&<>
            <div className="signin-or">{t("auth.signin.or")}</div>
            <button className="signin-btn signin-btn--google" onClick={startGoogle}><GoogleMark/> {t("auth.signin.google")}</button>
          </>}

          <div className="signin-footer">
            <hr/>
            <button className="admin-disclosure" onClick={()=>setAdmin(v=>!v)}><Lock/> {t("auth.action.admin_disclosure")} <ChevronDown className={admin?"rotate-180":""}/></button>
            {admin&&<form className="admin-login" onSubmit={passwordLogin}>{error&&<p>{error}</p>}<label><Mail/> <input type="email" value={email} onChange={e=>setEmail(e.target.value)} required/></label><label><Lock/> <input type="password" value={password} onChange={e=>setPassword(e.target.value)} required/></label><button>{t("auth.action.admin_sign_in")}</button></form>}
            <a className="signin-footer__help" href="/"><HelpCircle size={15}/> {t("auth.signin.help")}</a>
          </div>
        </>}
      </div>
    </section>
  </main>
}
