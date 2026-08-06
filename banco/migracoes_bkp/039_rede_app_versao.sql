BEGIN;

-- Versoes oficiais e links das lojas por rede (sobrescrevem configuracao global quando o app informa id_rede).
CREATE TABLE IF NOT EXISTS rede_app_versao (
  rede_id UUID NOT NULL PRIMARY KEY REFERENCES redes (id) ON DELETE CASCADE,
  versao_ios TEXT NOT NULL DEFAULT '0.0.0',
  versao_android TEXT NOT NULL DEFAULT '0.0.0',
  url_loja_ios TEXT NOT NULL DEFAULT '',
  url_loja_android TEXT NOT NULL DEFAULT '',
  mensagem_atualizacao TEXT NOT NULL DEFAULT '',
  atualizacao_obrigatoria BOOLEAN NOT NULL DEFAULT FALSE,
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rede_app_versao_rede_id ON rede_app_versao (rede_id);

ALTER TABLE rede_app_versao DISABLE ROW LEVEL SECURITY;

COMMENT ON TABLE rede_app_versao IS 'Versoes oficiais iOS/Android e URLs das lojas por rede; usado com GET /v1/app/versao?id_rede=...';

COMMIT;
