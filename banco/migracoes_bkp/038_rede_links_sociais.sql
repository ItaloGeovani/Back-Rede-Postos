-- Links de redes sociais configuráveis pelo gestor (exibidos no app quando app_modulo_redes_sociais).
BEGIN;

CREATE TABLE IF NOT EXISTS rede_links_sociais (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes (id) ON DELETE CASCADE,
  ordem SMALLINT NOT NULL DEFAULT 0 CHECK (ordem >= 0 AND ordem < 32),
  plataforma TEXT NOT NULL,
  titulo_exibicao TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_rede_links_sociais_rede_ordem
  ON rede_links_sociais (rede_id, ordem);

ALTER TABLE rede_links_sociais DISABLE ROW LEVEL SECURITY;

COMMIT;
