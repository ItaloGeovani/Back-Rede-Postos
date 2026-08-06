BEGIN;

-- Antes: pelo menos um canal (app OU posto). Agora: exatamente um (app XOR posto fisico).
ALTER TABLE campanhas DROP CONSTRAINT IF EXISTS chk_campanhas_canais_uso;

UPDATE campanhas
SET valida_no_posto_fisico = false
WHERE valida_no_app AND valida_no_posto_fisico;

UPDATE campanhas
SET valida_no_app = true, valida_no_posto_fisico = false
WHERE NOT valida_no_app AND NOT valida_no_posto_fisico;

ALTER TABLE campanhas
  ADD CONSTRAINT chk_campanhas_canais_uso
  CHECK (valida_no_app IS DISTINCT FROM valida_no_posto_fisico);

COMMENT ON COLUMN campanhas.valida_no_app IS 'TRUE somente se a promocao for exclusiva do app (valida_no_posto_fisico = FALSE)';
COMMENT ON COLUMN campanhas.valida_no_posto_fisico IS 'TRUE somente se a promocao for exclusiva do posto fisico (valida_no_app = FALSE)';

COMMIT;
