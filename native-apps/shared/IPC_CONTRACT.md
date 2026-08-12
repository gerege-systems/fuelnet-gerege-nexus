# Gerege Nexus native IPC

Canonical contract нь [`docs/SHELL_CONTRACT.md`](../../docs/SHELL_CONTRACT.md).
Swift, C# болон Kotlin bridge тус бүр document-start үед зөвхөн main-frame,
зөв origin-д `window.GeregeShell` v1.4-ийг inject хийнэ. Тэрхүү "зөв origin" нь
одоо тухайн төхөөрөмжийн domain шугам
([`device_lines.json`](device_lines.json)) — бүрхүүл бүр зөвхөн өөрийн шугамаас
ирсэн мессежийг хүлээн авна. Message envelope:

```json
{"id":"monotonic-string","method":"device.identity","params":{}}
```

Native reply нь `window.__geregeShellResolve(id, ok, JSONValue)` ганц entry
point-оор орно. Хуучин `GeregeNativeBridge`, `command/requestId`,
`navigate_path` протоколууд хүчингүй; шинэ код тэдгээрийг хэрэгжүүлэх ёсгүй.

Гүүрийн ямар ч method шинэ цонх нээхгүй. `shell.openPane` нь аппын ижил
хүрээн доторх дэлгэцийг л сольдог — гэрээний §1a.
