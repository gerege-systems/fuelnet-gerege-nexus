-- +goose Up

-- The AI copilot's product name becomes a placeholder, so a rename stops being
-- a migration.
--
-- Third time this row has been rewritten for a name. 00005 seeded "Gerege ERP
-- AI Copilot", 00012 made it "Gerege Nexus" at the rebrand, and the Gerege
-- Salus fork carried a fourth copy of the same UPDATE to say "Gerege Salus" —
-- a distribution that differed by a word and needed SQL to say so. The Go
-- default now substitutes {brand} with BRAND_NAME when it assembles the system
-- prompt (internal/platform/ai/copilot.go), so this is the last time.
--
-- The predicate is as narrow as 00012's and for the same reason: it matches
-- only the global row (tenant_id IS NULL) still holding byte-for-byte what
-- 00012 wrote. A tenant's own prompt, or a global one an operator has edited,
-- fails the comparison and is left alone. We are correcting our own default,
-- not overwriting somebody's decision — and an operator who wants the name in
-- their text can write {brand} themselves and keep it through the next rename.
--
-- Nothing about the safety clauses changes: answer only from approved tools,
-- never invent database values, never cross the tenant boundary. They travel
-- verbatim.

UPDATE ai_prompts
   SET content = 'You are {brand} AI Copilot. Answer only about platform operations and the information returned by approved tools. Never invent database values or expose another tenant''s data.',
       updated_at = NOW()
 WHERE tenant_id IS NULL
   AND prompt_key = 'scope'
   AND content = 'You are Gerege Nexus AI Copilot. Answer only about platform operations and the information returned by approved tools. Never invent database values or expose another tenant''s data.';

-- +goose Down

-- Symmetrical and just as narrow: put the literal name back only where the row
-- still holds exactly what Up wrote, so a rollback cannot discard newer work.
-- On a deployment running under another name this restores "Gerege Nexus",
-- which is what the row said before Up — a rollback returns the state it found,
-- including the reason this migration exists.

UPDATE ai_prompts
   SET content = 'You are Gerege Nexus AI Copilot. Answer only about platform operations and the information returned by approved tools. Never invent database values or expose another tenant''s data.',
       updated_at = NOW()
 WHERE tenant_id IS NULL
   AND prompt_key = 'scope'
   AND content = 'You are {brand} AI Copilot. Answer only about platform operations and the information returned by approved tools. Never invent database values or expose another tenant''s data.';
