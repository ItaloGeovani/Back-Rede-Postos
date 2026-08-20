-- Garante colunas usadas pelo painel (presença app + nível). Idempotente.
-- Clientes novos podem ter email NULL; queries devem usar COALESCE.

BEGIN;

ALTER TABLE usuarios
  ADD COLUMN IF NOT EXISTS ultimo_app_acesso_em TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS ultimo_app_plataforma TEXT;

ALTER TABLE usuarios
  ADD COLUMN IF NOT EXISTS nivel_cliente TEXT;

UPDATE usuarios
SET nivel_cliente = 'bronze'
WHERE nivel_cliente IS NULL OR TRIM(nivel_cliente) = '';

ALTER TABLE usuarios
  ALTER COLUMN nivel_cliente SET DEFAULT 'bronze';

DO $$
BEGIN
  ALTER TABLE usuarios ALTER COLUMN nivel_cliente SET NOT NULL;
EXCEPTION
  WHEN others THEN NULL;
END $$;

COMMENT ON COLUMN usuarios.ultimo_app_acesso_em IS
  'Ultimo heartbeat do app cliente (presenca); indica atividade recente.';
COMMENT ON COLUMN usuarios.ultimo_app_plataforma IS
  'Plataforma informada pelo app (ex.: android, ios, web).';
COMMENT ON COLUMN usuarios.nivel_cliente IS
  'Codigo do nivel (ex. bronze) alinhado a rede_niveis_cliente.';

CREATE INDEX IF NOT EXISTS idx_usuarios_rede_cliente_app_acesso
  ON usuarios (rede_id, ultimo_app_acesso_em DESC NULLS LAST)
  WHERE papel = 'cliente'::papel_usuario;

CREATE INDEX IF NOT EXISTS idx_usuarios_rede_nivel_cliente
  ON usuarios (rede_id, nivel_cliente)
  WHERE papel = 'cliente'::papel_usuario;

COMMIT;
