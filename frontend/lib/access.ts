"use client";

import { useEffect, useMemo, useState } from "react";
import { api } from "@/lib/api";

export interface Identity {
  id: string;
  is_admin: boolean;
  permissions?: string[];
}

/**
 * The caller's effective grant, as decided by the tenant's roles. One request
 * per page load: the promise is cached at module level so several screens (or
 * several components on one screen) share the same answer.
 *
 * This only decides what to render. Every endpoint re-checks the permission on
 * the server, so hiding a button is convenience, never the control itself.
 */
let identity: Promise<Identity> | null = null;

export function useAccess() {
  const [me, setMe] = useState<Identity | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let alive = true;
    identity ??= api.getMe();
    identity
      .then((value) => alive && setMe(value))
      // A failure here means the session is gone; Layout redirects to /login.
      .catch(() => undefined)
      .finally(() => alive && setReady(true));
    return () => {
      alive = false;
    };
  }, []);

  const granted = useMemo(() => new Set(me?.permissions || []), [me]);

  return {
    ready,
    isAdmin: me?.is_admin === true,
    can: (code: string) => me?.is_admin === true || granted.has(code),
  };
}
