import { redirect } from "next/navigation";

/**
 * Where PDF signing used to live.
 *
 * The screen moved when the E-Sign app was absorbed into Documents: one app has
 * one slug, and every one of its screens hangs off `/module/documents/*`. The
 * page itself did not change — only its address did.
 *
 * This stays because an address people have bookmarked, pinned to a kiosk, or
 * written into a device line's home screen is not ours to invalidate for a
 * reorganisation they did not ask for. A permanent redirect says so: the old
 * address is not gone, it has a forwarding note.
 *
 * The API kept its own prefix (`/api/v1/esign`) for the same reason, and needed
 * no note — see documents.RegisterRoutes.
 */
export default function ESignMoved() {
  redirect("/module/documents/pdf");
}
