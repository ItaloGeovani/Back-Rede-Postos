BEGIN;

ALTER TABLE campanhas
  ADD COLUMN IF NOT EXISTS imagem_url TEXT,
  ADD COLUMN IF NOT EXISTS titulo TEXT,
  ADD COLUMN IF NOT EXISTS posto_id UUID REFERENCES postos(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS vigencia_inicio TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS vigencia_fim TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_campanhas_rede_posto ON campanhas(rede_id, posto_id);
CREATE INDEX IF NOT EXISTS idx_campanhas_vigencia ON campanhas(rede_id, vigencia_inicio, vigencia_fim);

COMMENT ON COLUMN campanhas.posto_id IS 'NULL = promocao valida em todos os postos da rede; senao apenas no posto indicado';

COMMIT;
