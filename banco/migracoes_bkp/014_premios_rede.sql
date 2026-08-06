BEGIN;

CREATE TABLE IF NOT EXISTS premios (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id) ON DELETE CASCADE,
  titulo TEXT NOT NULL,
  imagem_url TEXT,
  valor_moeda NUMERIC(18, 4) NOT NULL CHECK (valor_moeda > 0),
  ativo BOOLEAN NOT NULL DEFAULT TRUE,
  vigencia_inicio TIMESTAMPTZ NOT NULL,
  vigencia_fim TIMESTAMPTZ,
  quantidade_disponivel INT CHECK (quantidade_disponivel IS NULL OR quantidade_disponivel >= 0),
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_premios_vigencia CHECK (vigencia_fim IS NULL OR vigencia_fim >= vigencia_inicio)
);

CREATE INDEX IF NOT EXISTS idx_premios_rede_vigencia ON premios(rede_id, vigencia_inicio DESC);

COMMENT ON TABLE premios IS 'Catalogo de premios resgataveis com moeda virtual da rede';
COMMENT ON COLUMN premios.quantidade_disponivel IS 'NULL = estoque ilimitado; 0 = esgotado ate reposicao';

ALTER TABLE premios DISABLE ROW LEVEL SECURITY;

COMMIT;
