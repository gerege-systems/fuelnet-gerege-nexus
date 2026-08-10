"use client";

import { useActionState } from "react";
import {
  registerPublisherAction,
  reviewAction,
  saveAppAction,
  submitVersionAction,
  type ActionState,
} from "./actions";

/**
 * The console's forms.
 *
 * Client components only so that the result of a submission can be shown in
 * place; everything they call is a server action, so the identity token stays
 * in the httpOnly cookie and the browser never holds it. Without JavaScript
 * these still post and still work — the answer arrives as a fresh page instead
 * of in the banner.
 */

function Banner({ state }: { state: ActionState }) {
  if (!state) return null;
  if (state.error) return <div className="bad">{state.error}</div>;
  return <div className="ok">{state.ok}</div>;
}

export function PublisherForm() {
  const [state, action, pending] = useActionState(registerPublisherAction, null);
  return (
    <form className="stack" action={action}>
      <Banner state={state} />
      <label>
        Байгууллагын нэр
        <input name="name" required placeholder="Жишээ ХХК" />
      </label>
      <label>
        Slug (жижиг үсэг, тоо, - ба _)
        <input name="slug" required pattern="[a-z0-9_\-]+" placeholder="jishee" />
      </label>
      <label>
        Холбоо барих и-мэйл
        <input name="contact_email" type="email" placeholder="dev@jishee.mn" />
      </label>
      <button className="primary" type="submit" disabled={pending}>
        {pending ? "Илгээж байна..." : "Publisher бүртгүүлэх"}
      </button>
    </form>
  );
}

export function AppForm({ app }: { app?: { id: string; slug: string; type: string; name: string; description: string; category: string; icon_url?: string } }) {
  const [state, action, pending] = useActionState(saveAppAction, null);
  return (
    <form className="stack" action={action}>
      <Banner state={state} />
      <label>
        Аппын ID (reverse-DNS, өөрчлөгдөхгүй)
        <input name="id" required defaultValue={app?.id} placeholder="mn.jishee.hrms" />
      </label>
      <label>
        Slug
        <input name="slug" required pattern="[a-z0-9_\-]+" defaultValue={app?.slug} placeholder="hrms" />
      </label>
      <label>
        Төрөл
        <select name="type" defaultValue={app?.type || "external"}>
          <option value="external">external — өөрийн сервер дээр ажилладаг платформ</option>
          <option value="module">module — платформын бинарт компиллогдсон модуль</option>
        </select>
      </label>
      <label>
        Нэр
        <input name="name" required defaultValue={app?.name} placeholder="Жишээ HRMS" />
      </label>
      <label>
        Тайлбар
        <input name="description" defaultValue={app?.description} />
      </label>
      <label>
        Ангилал
        <input name="category" defaultValue={app?.category} placeholder="Хүний нөөц" />
      </label>
      <label>
        Icon URL (заавал биш)
        <input name="icon_url" defaultValue={app?.icon_url} />
      </label>
      <button className="primary" type="submit" disabled={pending}>
        {pending ? "Хадгалж байна..." : "Хадгалах"}
      </button>
    </form>
  );
}

export function VersionForm({ slug, template }: { slug: string; template: string }) {
  const [state, action, pending] = useActionState(submitVersionAction, null);
  return (
    <form className="stack" action={action} style={{ maxWidth: "100%" }}>
      <Banner state={state} />
      <input type="hidden" name="slug" value={slug} />
      <label>
        Суваг
        <select name="channel" defaultValue="stable">
          <option value="stable">stable</option>
          <option value="beta">beta</option>
        </select>
      </label>
      <label>
        Manifest (JSON)
        {/* The manifest is the whole submission: version, platform constraint,
            permissions, menus and — for an external app — the launch URL and
            SSO client. The registry validates it with the same code a Nexus
            instance uses on arrival, so what is accepted here is what will be
            accepted there. */}
        <textarea name="manifest" required defaultValue={template} spellCheck={false} />
      </label>
      <button className="primary" type="submit" disabled={pending}>
        {pending ? "Илгээж байна..." : "Хянуулахаар илгээх"}
      </button>
    </form>
  );
}

export function ReviewForm({ id }: { id: string }) {
  const [state, action, pending] = useActionState(reviewAction, null);
  return (
    <form className="stack" action={action} style={{ maxWidth: "100%" }}>
      <Banner state={state} />
      <input type="hidden" name="id" value={id} />
      <label>
        Тэмдэглэл
        <input name="note" placeholder="Шийдвэрийн шалтгаан" />
      </label>
      <div className="split">
        <button className="primary" type="submit" name="action" value="publish" disabled={pending}>
          Нийтлэх
        </button>
        <button className="ghost" type="submit" name="action" value="reject" disabled={pending}>
          Татгалзах
        </button>
        <button className="ghost" type="submit" name="action" value="yank" disabled={pending}>
          Буцаан татах
        </button>
      </div>
    </form>
  );
}
