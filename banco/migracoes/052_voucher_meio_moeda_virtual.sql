-- Meio moeda virtual (rede ∩ posto) + colunas de débito em voucher_compras.
-- Chave JSON gateway_meios_habilitados.moeda_virtual (default false no parse da API).
-- Débito de moeda é imediato e sem estorno se o restante (PIX/dinheiro) expirar.

ALTER TABLE voucher_compras
  ADD COLUMN IF NOT EXISTS valor_moeda_fiat NUMERIC(14, 2) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS valor_moeda_token NUMERIC(18, 6) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS moeda_debitada_em TIMESTAMPTZ;

COMMENT ON COLUMN voucher_compras.valor_moeda_fiat IS
  'Valor em R$ coberto por moeda virtual (débito na criação). Sem estorno se restante expirar.';
COMMENT ON COLUMN voucher_compras.valor_moeda_token IS
  'Quantidade de tokens debitados da carteira do cliente na criação.';
COMMENT ON COLUMN voucher_compras.moeda_debitada_em IS
  'Quando a moeda foi debitada (NULL se não usou moeda).';

-- Amplia CHECK de meio_pagamento para MOEDA_VIRTUAL (100% moeda).
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'voucher_compras_meio_pagamento_check'
  ) THEN
    ALTER TABLE voucher_compras DROP CONSTRAINT voucher_compras_meio_pagamento_check;
  END IF;
  ALTER TABLE voucher_compras
    ADD CONSTRAINT voucher_compras_meio_pagamento_check
    CHECK (meio_pagamento IN ('PIX', 'DINHEIRO', 'MOEDA_VIRTUAL'));
END $$;

COMMENT ON COLUMN voucher_compras.meio_pagamento IS
  'PIX | DINHEIRO | MOEDA_VIRTUAL. Em misto (moeda+PIX/dinheiro): meio do restante + valor_moeda_* > 0.';
