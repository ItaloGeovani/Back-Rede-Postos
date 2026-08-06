BEGIN;

-- Faixa de litros (campanha por litro) e vínculo com combustíveis do catálogo da rede.
ALTER TABLE campanhas
  ADD COLUMN IF NOT EXISTS litros_min NUMERIC(14, 3),
  ADD COLUMN IF NOT EXISTS litros_max NUMERIC(14, 3);

COMMENT ON COLUMN campanhas.litros_min IS 'Mínimo de litros (inclusive) para campanha com base LITRO; NULL se não for por litro';
COMMENT ON COLUMN campanhas.litros_max IS 'Máximo de litros (inclusive) para campanha com base LITRO; NULL se não for por litro';

CREATE TABLE IF NOT EXISTS campanha_combustiveis_rede (
  campanha_id UUID NOT NULL REFERENCES campanhas(id) ON DELETE CASCADE,
  combustivel_rede_id UUID NOT NULL REFERENCES rede_combustiveis(id) ON DELETE RESTRICT,
  PRIMARY KEY (campanha_id, combustivel_rede_id)
);

CREATE INDEX IF NOT EXISTS idx_campanha_combustiveis_comb
  ON campanha_combustiveis_rede (combustivel_rede_id);

COMMIT;
