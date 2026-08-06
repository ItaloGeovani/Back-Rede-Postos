-- Quem registrou o uso do voucher no posto (baixa antes do abastecimento).
ALTER TABLE voucher_compras
  ADD COLUMN IF NOT EXISTS operador_usuario_id UUID REFERENCES usuarios (id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS operador_papel TEXT,
  ADD COLUMN IF NOT EXISTS operador_nome_snapshot TEXT;

CREATE INDEX IF NOT EXISTS idx_voucher_compras_operador
  ON voucher_compras (operador_usuario_id)
  WHERE operador_usuario_id IS NOT NULL;

COMMENT ON COLUMN voucher_compras.operador_usuario_id IS 'Frentista/gerente/gestor que deu baixa no voucher (marca como USADO).';
COMMENT ON COLUMN voucher_compras.operador_papel IS 'Papel do operador no momento da baixa (snapshot).';
COMMENT ON COLUMN voucher_compras.operador_nome_snapshot IS 'Nome do operador no momento da baixa (auditoria).';
