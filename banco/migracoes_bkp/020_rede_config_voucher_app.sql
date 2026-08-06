-- Configuração por rede: prazos de voucher comprado pelo app (PIX + uso no posto).

ALTER TABLE redes
  ADD COLUMN IF NOT EXISTS voucher_dias_validade_resgate int NOT NULL DEFAULT 7
    CHECK (voucher_dias_validade_resgate >= 1 AND voucher_dias_validade_resgate <= 365);

ALTER TABLE redes
  ADD COLUMN IF NOT EXISTS voucher_minutos_expira_pagamento_pix int NOT NULL DEFAULT 30
    CHECK (voucher_minutos_expira_pagamento_pix >= 5 AND voucher_minutos_expira_pagamento_pix <= 10080);
