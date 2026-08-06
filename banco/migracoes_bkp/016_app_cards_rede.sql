BEGIN;

-- Cards do app: slot 0 = destaque da rede; 1..3 = promocoes (imagens/links configuraveis no painel).
CREATE TABLE IF NOT EXISTS app_cards_rede (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id) ON DELETE CASCADE,
  slot SMALLINT NOT NULL CHECK (slot >= 0 AND slot <= 3),
  titulo TEXT NOT NULL DEFAULT '',
  imagem_url TEXT NOT NULL DEFAULT '',
  link_url TEXT NOT NULL DEFAULT '',
  ativo BOOLEAN NOT NULL DEFAULT TRUE,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rede_id, slot)
);

CREATE INDEX IF NOT EXISTS idx_app_cards_rede_rede ON app_cards_rede(rede_id);

COMMENT ON TABLE app_cards_rede IS 'Cards do app mobile por rede: slot 0 destaque, 1-3 promocoes';
COMMENT ON COLUMN app_cards_rede.slot IS '0 = card destaque rede; 1,2,3 = promocoes';

ALTER TABLE app_cards_rede DISABLE ROW LEVEL SECURITY;

COMMIT;
