"use client";

import { useEffect, useState } from "react";
import { getShell, SHELL_EVENTS, SHELL_METHODS } from "@/lib/shell";

type Check = { name: string; ok: boolean; detail: string };

export default function ShellConformancePage() {
  const [checks, setChecks] = useState<Check[]>([]);
  const [eventPayload, setEventPayload] = useState("event хүлээгээгүй");

  useEffect(() => {
    const shell = getShell();
    if (!shell) {
      setChecks([{ name: "Browser isolation", ok: true, detail: "GeregeShell байхгүй — PWA горим" }]);
      return;
    }
    setChecks([
      { name: "Contract", ok: /^1\.[1-3](?:\.|$)/.test(shell.version), detail: shell.version },
      { name: "Frozen object", ok: Object.isFrozen(shell), detail: String(Object.isFrozen(shell)) },
      { name: "Platform", ok: ["macos", "windows", "ios", "android", "kiosk", "pos"].includes(shell.platform), detail: shell.platform },
      { name: "Form factor", ok: ["desktop", "mobile", "tablet", "kiosk", "pos"].includes(shell.formFactor), detail: shell.formFactor },
      { name: "invoke", ok: typeof shell.invoke === "function", detail: typeof shell.invoke },
      { name: "on/unsubscribe", ok: typeof shell.on === "function", detail: typeof shell.on },
    ]);
    return shell.on(SHELL_EVENTS.AUTH_CHANGED, payload => setEventPayload(JSON.stringify(payload)));
  }, []);

  async function proveUnknownMethodRejects() {
    const shell = getShell();
    if (!shell) return;
    let ok = false;
    try { await shell.invoke("conformance.unknown"); } catch { ok = true; }
    setChecks(current => [...current.filter(c => c.name !== "Unknown method"), { name: "Unknown method", ok, detail: ok ? "rejected" : "алдаатайгаар resolved" }]);
  }

  return <main style={{ maxWidth: 760, margin: "48px auto", padding: 24 }}>
    <h1>Gerege Shell conformance</h1>
    <p>Native bridge-ийн contract, browser isolation, event ба reject зан төлөвийг шалгана.</p>
    <ul>{checks.map(check => <li key={check.name}>{check.ok ? "✅" : "❌"} {check.name}: <code>{check.detail}</code></li>)}</ul>
    <p>auth event: <code>{eventPayload}</code></p>
    <button onClick={proveUnknownMethodRejects}>Unknown method шалгах</button>{" "}
    <button onClick={() => void getShell()?.invoke(SHELL_METHODS.AUTH_LOCK)}>Native lock шалгах</button>
  </main>;
}
