-- Expiração opcional de moeda virtual por lote (crédito).
-- moeda_virtual_expira_dias: NULL/0 = não expira; >0 = dias para cada crédito novo.

ALTER TABLE redes
  ADD COLUMN IF NOT EXISTS moeda_virtual_expira_dias INT NULL
    CHECK (moeda_virtual_expira_dias IS NULL OR (moeda_virtual_expira_dias >= 0 AND moeda_virtual_expira_dias <= 365));

COMMENT ON COLUMN redes.moeda_virtual_expira_dias IS
  'Dias para expirar cada crédito novo de moeda (CASHBACK/BONUS). 0 ou NULL = sem expiração.';

DO $$
BEGIN
  ALTER TYPE tipo_transacao_carteira ADD VALUE IF NOT EXISTS 'EXPIRACAO';
EXCEPTION
  WHEN duplicate_object THEN NULL;
  WHEN undefined_object THEN NULL;
END $$;

-- Fallback para Postgres sem IF NOT EXISTS em enum:
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_enum e
    JOIN pg_type t ON t.oid = e.enumtypid
    WHERE t.typname = 'tipo_transacao_carteira' AND e.enumlabel = 'EXPIRACAO'
  ) THEN
    ALTER TYPE tipo_transacao_carteira ADD VALUE 'EXPIRACAO';
  END IF;
EXCEPTION
  WHEN others THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS carteira_lotes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  carteira_id UUID NOT NULL REFERENCES carteiras(id),
  transacao_id UUID REFERENCES transacoes_carteira(id),
  valor_inicial NUMERIC(18, 6) NOT NULL CHECK (valor_inicial > 0),
  valor_restante NUMERIC(18, 6) NOT NULL CHECK (valor_restante >= 0),
  origem_tipo TEXT NOT NULL,
  tipo_referencia TEXT NOT NULL DEFAULT '',
  referencia_id UUID,
  creditado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expira_em TIMESTAMPTZ,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_carteira_lotes_restante_le_inicial CHECK (valor_restante <= valor_inicial)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_carteira_lotes_transacao
  ON carteira_lotes (transacao_id)
  WHERE transacao_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_carteira_lotes_ativos
  ON carteira_lotes (carteira_id, expira_em NULLS FIRST, creditado_em)
  WHERE valor_restante > 0;

CREATE INDEX IF NOT EXISTS idx_carteira_lotes_expira
  ON carteira_lotes (rede_id, expira_em)
  WHERE valor_restante > 0 AND expira_em IS NOT NULL;

COMMENT ON TABLE carteira_lotes IS
  'Lotes de crédito de moeda; gastos consomem FIFO; expira_em NULL = não expira.';

-- Backfill: um lote sem expiração por carteira com saldo > 0 (preserva saldo atual).
INSERT INTO carteira_lotes (
  rede_id, carteira_id, transacao_id, valor_inicial, valor_restante,
  origem_tipo, tipo_referencia, creditado_em, expira_em
)
SELECT
  c.rede_id,
  c.id,
  NULL,
  s.saldo,
  s.saldo,
  'LEGADO',
  'backfill_saldo',
  NOW(),
  NULL
FROM carteiras c
INNER JOIN LATERAL (
  SELECT COALESCE(SUM(t.valor_token * t.direcao), 0)::numeric AS saldo
  FROM transacoes_carteira t
  WHERE t.carteira_id = c.id AND t.rede_id = c.rede_id
) s ON true
WHERE s.saldo > 0
  AND NOT EXISTS (
    SELECT 1 FROM carteira_lotes l WHERE l.carteira_id = c.id AND l.valor_restante > 0
  );
