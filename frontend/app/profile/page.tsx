"use client";
import {useEffect,useState} from "react";
import {api} from "@/lib/api";
import {useI18n} from "@/lib/i18n";
import {Building2,Fingerprint,KeyRound,MonitorSmartphone,ShieldCheck} from "lucide-react";

/**
 * Хүний өөрийнх нь тухай бичлэг.
 *
 * Платформын дэлгэц, суулгадаг апп биш. Апп нь байгууллага тус бүрд суудаг
 * бөгөөд админ нь устгаж чадна — хүн ямар таних тэмдгээр нэвтэрдгээ харах
 * эрхийг ажил олгогч нь авч болдог байх нь буруу. Мөн олон байгууллагад
 * харьяалагдах хүнд нэг л профайл байна, гишүүнчлэл тутамд нэг биш.
 */

type Identity = {
  kind: string; provider: string; subject: string;
  email?: string; name?: string; surname?: string;
  linked_at: string; last_seen_at: string;
  claims?: Record<string, unknown>;
};
type Profile = {
  id: string; name: string; email: string; created_at: string; is_admin: boolean;
  organisations: Array<{id:string;name:string;slug:string}>;
  identities: Identity[]; active_sessions: number;
};

function when(iso: string) {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "—" : d.toLocaleDateString();
}

export default function ProfilePage(){const {t}=useI18n();
  const [profile,setProfile]=useState<Profile|null>(null);
  const [error,setError]=useState("");
  const [open,setOpen]=useState<string>("");

  useEffect(()=>{void api.profile().then(setProfile).catch((e:any)=>setError(e?.message||"—"))},[]);

  if(error)return <main className="profile"><p className="profile__error">{error}</p></main>;
  if(!profile)return <main className="profile"><p className="profile__muted">{t("profile.loading")}</p></main>;

  return <main className="profile">
    <header className="profile__head">
      <div className="profile__avatar">{(profile.name||profile.email||"?").trim().charAt(0).toUpperCase()}</div>
      <div>
        <h1>{profile.name||profile.email}</h1>
        <p>{profile.email}</p>
      </div>
    </header>

    {/* Тойм: тоо биш, хариулт. Хэдэн байгууллагад, хэдэн аргаар нэвтэрдэг,
        хаана нээлттэй байна. */}
    <section className="profile__stats">
      <div><Building2/><b>{profile.organisations.length}</b><span>{t("profile.stat.organisations")}</span></div>
      <div><KeyRound/><b>{profile.identities.length}</b><span>{t("profile.stat.identities")}</span></div>
      <div><MonitorSmartphone/><b>{profile.active_sessions}</b><span>{t("profile.stat.sessions")}</span></div>
      <div><ShieldCheck/><b>{when(profile.created_at)}</b><span>{t("profile.stat.since")}</span></div>
    </section>

    <section className="profile__section">
      <h2>{t("profile.identities")}</h2>
      <p className="profile__muted">{t("profile.identities_lede")}</p>
      <ul className="profile__list">
        {profile.identities.map(id=>{
          const key=id.kind+id.subject;
          const claims=Object.entries(id.claims||{});
          return <li key={key}>
            <div className="profile__row">
              <span className="profile__icon">{id.kind==="eid"?<Fingerprint/>:<KeyRound/>}</span>
              <div className="profile__grow">
                <b>{id.provider}</b>
                <span>{id.email||[id.surname,id.name].filter(Boolean).join(" ")||id.subject}</span>
              </div>
              <div className="profile__meta">
                <span>{t("profile.linked_at")} {when(id.linked_at)}</span>
                <span>{t("profile.last_seen")} {when(id.last_seen_at)}</span>
              </div>
            </div>
            {claims.length>0&&<>
              <button className="profile__toggle" onClick={()=>setOpen(open===key?"":key)}>
                {open===key?t("profile.hide_claims"):t("profile.show_claims",{count:String(claims.length)})}
              </button>
              {open===key&&<dl className="profile__claims">
                {claims.map(([k,v])=><div key={k}><dt>{k}</dt><dd>{typeof v==="object"?JSON.stringify(v):String(v)}</dd></div>)}
              </dl>}
            </>}
          </li>;
        })}
        {profile.identities.length===0&&<li className="profile__muted">{t("profile.no_identities")}</li>}
      </ul>
    </section>

    <section className="profile__section">
      <h2>{t("profile.organisations")}</h2>
      <ul className="profile__list">
        {profile.organisations.map(o=><li key={o.id}><div className="profile__row">
          <span className="profile__icon"><Building2/></span>
          <div className="profile__grow"><b>{o.name}</b><span>{o.slug}</span></div>
        </div></li>)}
      </ul>
    </section>
  </main>
}
