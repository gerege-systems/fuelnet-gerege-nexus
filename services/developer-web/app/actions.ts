"use server";

import { revalidatePath } from "next/cache";
import { createPublisher, decideVersion, submitVersion, upsertApp } from "@/lib/registry";

/**
 * Everything the console can change, as server actions.
 *
 * Forms post straight to these, so the console works without JavaScript and no
 * page ever holds the identity token. Each returns a message rather than
 * throwing: a publisher who mistypes a manifest needs to be told what the
 * registry said about it, not shown a stack trace.
 */

export type ActionState = { ok?: string; error?: string } | null;

function text(form: FormData, field: string) {
  return String(form.get(field) ?? "").trim();
}

export async function registerPublisherAction(_state: ActionState, form: FormData): Promise<ActionState> {
  const result = await createPublisher({
    slug: text(form, "slug"),
    name: text(form, "name"),
    contact_email: text(form, "contact_email"),
  });
  if ("error" in result) return { error: result.error };
  revalidatePath("/");
  return { ok: `Publisher "${result.data.name}" бүртгэгдлээ.` };
}

export async function saveAppAction(_state: ActionState, form: FormData): Promise<ActionState> {
  const result = await upsertApp({
    id: text(form, "id"),
    slug: text(form, "slug"),
    type: text(form, "type") || "module",
    name: text(form, "name"),
    description: text(form, "description"),
    category: text(form, "category"),
    icon_url: text(form, "icon_url"),
    visibility: "public",
  });
  if ("error" in result) return { error: result.error };
  revalidatePath("/");
  return { ok: `"${result.data.name}" хадгалагдлаа.` };
}

export async function submitVersionAction(_state: ActionState, form: FormData): Promise<ActionState> {
  const slug = text(form, "slug");
  let manifest: unknown;
  try {
    manifest = JSON.parse(text(form, "manifest"));
  } catch (cause) {
    // Parsed here rather than posted raw, so a missing brace is answered
    // immediately instead of travelling to the registry to come back as a
    // decoding error about a request body.
    return { error: `Manifest JSON алдаатай: ${(cause as Error).message}` };
  }

  const result = await submitVersion(slug, { channel: text(form, "channel") || "stable", manifest });
  if ("error" in result) return { error: result.error };
  revalidatePath(`/apps/${slug}`);
  return { ok: `Хувилбар ${result.data.version} хянуулахаар илгээгдлээ.` };
}

export async function reviewAction(_state: ActionState, form: FormData): Promise<ActionState> {
  const result = await decideVersion(text(form, "id"), text(form, "action"), text(form, "note"));
  if ("error" in result) return { error: result.error };
  revalidatePath("/review");
  revalidatePath("/");
  return { ok: `Шийдвэр бичигдлээ: ${result.data.status}.` };
}
