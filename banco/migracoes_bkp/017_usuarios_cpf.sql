-- =============================================================================
-- 017 — Coluna cpf em usuarios (cadastro do app cliente / PIX futuro)
--
-- A tabela `usuarios` criada em 001 nao tinha CPF; esta migracao adiciona.
--
-- Aplicar com o projeto ja configurado (.env com Postgres):
--   cd servidor-go
--   go run ./cmd/aplicar-migracoes
--
-- Conferir no psql (deve retornar 1 linha):
--   SELECT column_name, data_type
--   FROM information_schema.columns
--   WHERE table_schema = 'public' AND table_name = 'usuarios' AND column_name = 'cpf';
-- =============================================================================

BEGIN;

ALTER TABLE usuarios
  ADD COLUMN IF NOT EXISTS cpf TEXT;

COMMENT ON COLUMN usuarios.cpf IS 'CPF do usuario (somente digitos); uso futuro ex. PIX.';

CREATE UNIQUE INDEX IF NOT EXISTS uq_usuarios_rede_cpf
  ON usuarios (rede_id, cpf)
  WHERE cpf IS NOT NULL AND TRIM(cpf) <> '';

COMMIT;
