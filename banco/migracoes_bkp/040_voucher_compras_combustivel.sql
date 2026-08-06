-- Combustível escolhido na compra por litro (exibe para o frentista no resgate).
ALTER TABLE voucher_compras
  ADD COLUMN IF NOT EXISTS combustivel_rede_id UUID REFERENCES rede_combustiveis (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_voucher_compras_combustivel_rede
  ON voucher_compras (combustivel_rede_id)
  WHERE combustivel_rede_id IS NOT NULL;

COMMENT ON COLUMN voucher_compras.combustivel_rede_id IS 'FK rede_combustiveis: produto na compra por litro; nulo fora desse fluxo.';
