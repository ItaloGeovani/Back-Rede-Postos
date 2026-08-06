-- Meio de pagamento DINHEIRO + status AGUARDANDO_DINHEIRO (código gerado na criação).

DO $$
BEGIN
  ALTER TYPE status_voucher_compra ADD VALUE 'AGUARDANDO_DINHEIRO';
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE voucher_compras
  ADD COLUMN IF NOT EXISTS meio_pagamento TEXT NOT NULL DEFAULT 'PIX';

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'voucher_compras_meio_pagamento_check'
  ) THEN
    ALTER TABLE voucher_compras
      ADD CONSTRAINT voucher_compras_meio_pagamento_check
      CHECK (meio_pagamento IN ('PIX', 'DINHEIRO'));
  END IF;
END $$;

COMMENT ON COLUMN voucher_compras.meio_pagamento IS 'PIX (gateway online) ou DINHEIRO (pagamento no posto ao frentista).';
