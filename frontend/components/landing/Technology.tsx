"use client";

import React from "react";
import {ArrowRight} from "lucide-react";

import {TECHNOLOGY} from "@/components/landing/content";
import {useI18n} from "@/lib/i18n";
import {useBrand} from "@/lib/brandContext";

/**
 * The three parts a sign-in passes through, separated by arrows.
 *
 * The arrows are laid out between the columns rather than inside them, which is
 * why this is built from a fragment per column instead of one element each —
 * `.gp-tech` styles its `div` children and its own `svg` children differently,
 * and the arrow belongs to the row, not to either side of it.
 */
export default function Technology() {
  const {t} = useI18n();
  const brand = useBrand();

  return (
    <section className="gp-tech" id="technology">
      {TECHNOLOGY.map(({icon: Icon, name, body}, index) => (
        <React.Fragment key={name}>
          {index > 0 && <ArrowRight />}
          <div>
            <Icon />
            <h3>{name.replace("{brand}", brand.name)}</h3>
            <p>{t(body)}</p>
          </div>
        </React.Fragment>
      ))}
    </section>
  );
}
