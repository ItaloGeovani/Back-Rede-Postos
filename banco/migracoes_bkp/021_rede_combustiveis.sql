-- Catálogo de combustíveis por rede (nome, preço por litro de referência).

CREATE TABLE IF NOT EXISTS rede_combustiveis (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes (id) ON DELETE CASCADE,
  nome TEXT NOT NULL,
  codigo TEXT,
  descricao TEXT,
  preco_por_litro NUMERIC(12, 4) NOT NULL CHECK (preco_por_litro >= 0),
  ativo BOOLEAN NOT NULL DEFAULT TRUE,
  ordem INT NOT NULL DEFAULT 0,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rede_combustiveis_rede
  ON rede_combustiveis (rede_id);

CREATE INDEX IF NOT EXISTS idx_rede_combustiveis_rede_ordem
  ON rede_combustiveis (rede_id, ordem, nome);

CREATE UNIQUE INDEX IF NOT EXISTS uq_rede_combustivel_codigo
  ON rede_combustiveis (rede_id, lower(trim(codigo)))
  WHERE codigo IS NOT NULL AND TRIM(codigo) <> '';

COMMENT ON TABLE rede_combustiveis IS 'Combustíveis da rede: referência de nome e preço por litro (gestor/gerente).';
