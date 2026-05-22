-- Modo de gateway: REDE (uma conta MP para todos os postos) ou POSTO (conta MP por unidade).
-- voucher_compras.posto_id_compra: posto escolhido na compra (modo POSTO); resgate restrito a esse posto.

BEGIN;

ALTER TABLE redes
  ADD COLUMN IF NOT EXISTS gateway_pagamento_modo TEXT NOT NULL DEFAULT 'REDE'
    CHECK (gateway_pagamento_modo IN ('REDE', 'POSTO'));

COMMENT ON COLUMN redes.gateway_pagamento_modo IS
  'REDE: PIX na conta MP da rede (rede_mercado_pago). POSTO: PIX na conta MP de cada posto; cliente escolhe posto na compra.';

CREATE TABLE IF NOT EXISTS posto_mercado_pago (
  posto_id UUID PRIMARY KEY REFERENCES postos (id) ON DELETE CASCADE,
  mp_access_token TEXT,
  mp_webhook_secret TEXT,
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE posto_mercado_pago IS 'Mercado Pago por posto (modo POSTO).';

ALTER TABLE voucher_compras
  ADD COLUMN IF NOT EXISTS posto_id_compra UUID REFERENCES postos (id);

CREATE INDEX IF NOT EXISTS idx_voucher_compras_posto_compra
  ON voucher_compras (rede_id, posto_id_compra)
  WHERE posto_id_compra IS NOT NULL;

COMMENT ON COLUMN voucher_compras.posto_id_compra IS
  'Posto escolhido na compra (modo POSTO). Resgate permitido somente neste posto. NULL = legado ou modo REDE.';

COMMIT;
