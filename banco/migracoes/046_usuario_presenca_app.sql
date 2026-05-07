BEGIN;

ALTER TABLE usuarios
  ADD COLUMN IF NOT EXISTS ultimo_app_acesso_em TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS ultimo_app_plataforma TEXT;

COMMENT ON COLUMN usuarios.ultimo_app_acesso_em IS
  'Ultimo heartbeat do app cliente (presenca); indica atividade recente, nao ligacao persistente.';
COMMENT ON COLUMN usuarios.ultimo_app_plataforma IS
  'Plataforma informada pelo app (ex.: android, ios, web).';

CREATE INDEX IF NOT EXISTS idx_usuarios_rede_cliente_app_acesso
  ON usuarios (rede_id, ultimo_app_acesso_em DESC NULLS LAST)
  WHERE papel = 'cliente'::papel_usuario;

COMMIT;
