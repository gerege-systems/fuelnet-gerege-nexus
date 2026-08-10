# Gerege Nexus native IPC

Canonical contract нь [`docs/SHELL_CONTRACT.md`](../../docs/SHELL_CONTRACT.md).
Swift, C# болон Kotlin bridge тус бүр document-start үед зөвхөн main-frame,
зөв origin-д `window.GeregeShell` v1.3-ыг inject хийнэ. Message envelope:

```json
{"id":"monotonic-string","method":"device.identity","params":{}}
```

Native reply нь `window.__geregeShellResolve(id, ok, JSONValue)` ганц entry
point-оор орно. Хуучин `GeregeNativeBridge`, `command/requestId`,
`navigate_path` протоколууд хүчингүй; шинэ код тэдгээрийг хэрэгжүүлэх ёсгүй.
