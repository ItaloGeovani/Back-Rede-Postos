BEGIN;

ALTER TABLE voucher_compras
  ADD COLUMN IF NOT EXISTS tipo_beneficio TEXT NOT NULL DEFAULT 'DESCONTO'
    CHECK (tipo_beneficio IN ('DESCONTO', 'CASHBACK')),
  ADD COLUMN IF NOT EXISTS cashback_percentual NUMERIC(8, 4),
  ADD COLUMN IF NOT EXISTS cashback_valor NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (cashback_valor >= 0),
  ADD COLUMN IF NOT EXISTS cashback_creditado_em TIMESTAMPTZ;

UPDATE voucher_compras
SET tipo_beneficio = 'DESCONTO'
WHERE tipo_beneficio IS NULL OR trim(tipo_beneficio) = '';

CREATE INDEX IF NOT EXISTS idx_voucher_compras_cashback_pendente
  ON voucher_compras (rede_id, status, cashback_creditado_em)
  WHERE tipo_beneficio = 'CASHBACK';

COMMENT ON COLUMN voucher_compras.tipo_beneficio IS
  'DESCONTO aplica abatimento no pagamento; CASHBACK credita moeda apos pagamento aprovado.';
COMMENT ON COLUMN voucher_compras.cashback_percentual IS
  'Percentual do cashback da campanha (ex.: 0.5 para 0,5%).';
COMMENT ON COLUMN voucher_compras.cashback_valor IS
  'Valor em R$ do cashback previsto/creditado para a compra.';

COMMIT;
