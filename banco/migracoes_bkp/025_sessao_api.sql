-- Sessões de API (tokens tok_*) com expiração em PostgreSQL: sobrevivem a restarts do processo.
CREATE TABLE IF NOT EXISTS sessao_api (
  token text PRIMARY KEY,
  usuario_id text NOT NULL,
  id_rede text NOT NULL,
  id_posto text NOT NULL DEFAULT '',
  nome_completo text NOT NULL DEFAULT '',
  papel text NOT NULL,
  expira_em timestamptz NOT NULL,
  criado_em timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessao_api_expira ON sessao_api (expira_em);
