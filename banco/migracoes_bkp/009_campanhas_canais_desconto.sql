BEGIN;

ALTER TABLE campanhas
  ADD COLUMN IF NOT EXISTS valida_no_app BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS valida_no_posto_fisico BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS modalidade_desconto TEXT NOT NULL DEFAULT 'NENHUM'
    CHECK (modalidade_desconto IN ('NENHUM', 'PERCENTUAL', 'VALOR_FIXO')),
  ADD COLUMN IF NOT EXISTS base_desconto TEXT NOT NULL DEFAULT 'VALOR_COMPRA'
    CHECK (base_desconto IN ('VALOR_COMPRA', 'LITRO', 'UNIDADE')),
  ADD COLUMN IF NOT EXISTS valor_desconto NUMERIC(14, 4) NOT NULL DEFAULT 0
    CHECK (valor_desconto >= 0),
  ADD COLUMN IF NOT EXISTS valor_minimo_compra NUMERIC(14, 2) NOT NULL DEFAULT 0
    CHECK (valor_minimo_compra >= 0),
  ADD COLUMN IF NOT EXISTS max_usos_por_cliente INT
    CHECK (max_usos_por_cliente IS NULL OR max_usos_por_cliente >= 1);

ALTER TABLE campanhas
  ADD CONSTRAINT chk_campanhas_canais_uso
  CHECK (valida_no_app OR valida_no_posto_fisico);

COMMENT ON COLUMN campanhas.valida_no_app IS 'Promocao pode ser usada no aplicativo';
COMMENT ON COLUMN campanhas.valida_no_posto_fisico IS 'Promocao vale no posto fisico (balcao/bomba)';
COMMENT ON COLUMN campanhas.modalidade_desconto IS 'NENHUM = so informativo; PERCENTUAL ou VALOR_FIXO quando ha desconto';
COMMENT ON COLUMN campanhas.base_desconto IS 'Sobre o total da compra, por litro ou por unidade de produto';
COMMENT ON COLUMN campanhas.max_usos_por_cliente IS 'NULL = ilimitado ate o fim da vigencia; 1 = uma vez por cliente; N = no maximo N usos por cliente';

COMMIT;
