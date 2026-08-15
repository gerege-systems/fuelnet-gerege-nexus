/**
 * The name, the mark and the colour this deployment answers to.
 *
 * `lib/apiBase.ts` took the deployment's *address* out of the build. This takes
 * its *identity* out, and the argument is the same one a step further: an image
 * that knows its own name is an image one customer owns. Level 1 of the
 * ecosystem strategy — one image, a different `.env`, a hundred deployments —
 * is only true when both are read at run time.
 *
 * Read from the process environment rather than from `NEXT_PUBLIC_*`, because a
 * NEXT_PUBLIC value is substituted into the bundle by `next build`: it would be
 * the same variable, baked at exactly the moment this file exists to stop
 * baking it. That is why `app/layout.tsx` reads it on the server and hands the
 * result down, and why it is dynamic — see the note there.
 *
 * The reading itself lives in `lib/brandEnv.ts`, apart from the shape and the
 * defaults here, so that a client component naming the product does not drag a
 * `process.env` reader into every browser bundle.
 *
 * `BRAND_NAME` is also read by the API (backend/internal/platform/config), for
 * the two sentences it puts in front of a person. They are two halves of one
 * setting; a deployment that renames only one of them has a sign-in screen and
 * an eID prompt that disagree about which product this is.
 *
 * Only the name is expected to be set. The rest exist because a rebrand that
 * changes the word and keeps the blue emblem is not a rebrand, and because the
 * values are all a deployment can supply without a build: an image cannot be
 * given a new logo file after the fact, but it can be pointed at one.
 */
export type Brand = {
  name: string;
  /** What a launcher has room for under an icon. */
  shortName: string;
  description: string;
  /** Where the mark is served from — a path on this host, or an absolute URL. */
  logoUrl: string;
  /** The browser and launcher chrome colour, as a CSS hex colour. */
  themeColor: string;
};

export const DEFAULT_BRAND: Brand = {
  name: "Gerege Nexus",
  shortName: "Nexus",
  description:
    "Төрийн болон хувийн хэвшлийн байгууллагын үйлчилгээ, үйл ажиллагаа, систем, өгөгдлийг нэгтгэх модульт платформ.",
  logoUrl: "/brand.webp",
  themeColor: "#1869eb",
};
