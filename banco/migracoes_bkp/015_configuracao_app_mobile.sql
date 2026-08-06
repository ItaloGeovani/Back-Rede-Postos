BEGIN;

CREATE TABLE IF NOT EXISTS configuracao_app_mobile (
  id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  versao_ios TEXT NOT NULL DEFAULT '0.0.0',
  versao_android TEXT NOT NULL DEFAULT '0.0.0',
  url_loja_ios TEXT NOT NULL DEFAULT '',
  url_loja_android TEXT NOT NULL DEFAULT '',
  mensagem_atualizacao TEXT NOT NULL DEFAULT '',
  atualizacao_obrigatoria BOOLEAN NOT NULL DEFAULT FALSE,
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO configuracao_app_mobile (id) VALUES (1)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE configuracao_app_mobile DISABLE ROW LEVEL SECURITY;

COMMENT ON TABLE configuracao_app_mobile IS 'Versoes oficiais iOS/Android e links das lojas; uma unica linha (id=1).';

COMMIT;
