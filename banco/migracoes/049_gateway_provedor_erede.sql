-- Provedor de pagamento exclusivo (MERCADO_PAGO | E_REDE), meios habilitados e credenciais e.Rede.

BEGIN;

ALTER TABLE redes
  ADD COLUMN IF NOT EXISTS gateway_provedor_ativo TEXT NOT NULL DEFAULT 'MERCADO_PAGO'
    CHECK (gateway_provedor_ativo IN ('MERCADO_PAGO', 'E_REDE'));

ALTER TABLE redes
  ADD COLUMN IF NOT EXISTS gateway_meios_habilitados JSONB NOT NULL DEFAULT '{"pix": true, "cartao_credito": false, "cartao_debito": false}'::jsonb;

COMMENT ON COLUMN redes.gateway_provedor_ativo IS 'Provedor unico ativo para cobrancas da rede.';
COMMENT ON COLUMN redes.gateway_meios_habilitados IS 'Meios aceitos: pix, cartao_credito, cartao_debito.';

CREATE TABLE IF NOT EXISTS rede_erede (
  rede_id UUID PRIMARY KEY REFERENCES redes (id) ON DELETE CASCADE,
  pv TEXT,
  client_secret TEXT,
  ambiente TEXT NOT NULL DEFAULT 'sandbox' CHECK (ambiente IN ('sandbox', 'producao')),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS posto_erede (
  posto_id UUID PRIMARY KEY REFERENCES postos (id) ON DELETE CASCADE,
  pv TEXT,
  client_secret TEXT,
  ambiente TEXT NOT NULL DEFAULT 'sandbox' CHECK (ambiente IN ('sandbox', 'producao')),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE voucher_compras
  ADD COLUMN IF NOT EXISTS gateway_provedor TEXT,
  ADD COLUMN IF NOT EXISTS gateway_tid TEXT;

CREATE INDEX IF NOT EXISTS idx_voucher_compras_gateway_tid
  ON voucher_compras (gateway_tid)
  WHERE gateway_tid IS NOT NULL;

COMMENT ON COLUMN voucher_compras.gateway_provedor IS 'MERCADO_PAGO ou E_REDE no momento da compra.';
COMMENT ON COLUMN voucher_compras.gateway_tid IS 'ID externo (tid e.Rede ou payment_id MP como texto).';

COMMIT;
