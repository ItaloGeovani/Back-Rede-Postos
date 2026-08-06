-- Voucher pré-pago comprado no app: valor, campanha opcional, pagamento (Mercado Pago) e resgate no posto.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'status_voucher_compra') THEN
    CREATE TYPE status_voucher_compra AS ENUM (
      'AGUARDANDO_PAGAMENTO',
      'ATIVO',
      'USADO',
      'EXPIRADO',
      'CANCELADO'
    );
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS voucher_compras (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes (id),
  usuario_id UUID NOT NULL REFERENCES usuarios (id),
  campanha_id UUID REFERENCES campanhas (id),
  valor_solicitado NUMERIC(12, 2) NOT NULL CHECK (valor_solicitado > 0),
  desconto_aplicado NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (desconto_aplicado >= 0),
  valor_final NUMERIC(12, 2) NOT NULL CHECK (valor_final > 0),
  status status_voucher_compra NOT NULL DEFAULT 'AGUARDANDO_PAGAMENTO',
  mp_payment_id BIGINT,
  referencia_pagamento TEXT,
  codigo_resgate TEXT UNIQUE,
  expira_pagamento_em TIMESTAMPTZ,
  expira_resgate_em TIMESTAMPTZ,
  usado_em TIMESTAMPTZ,
  posto_id_uso UUID REFERENCES postos (id),
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_voucher_compras_rede_user ON voucher_compras (rede_id, usuario_id, criado_em DESC);
CREATE INDEX IF NOT EXISTS idx_voucher_compras_codigo ON voucher_compras (codigo_resgate) WHERE codigo_resgate IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_voucher_compras_status ON voucher_compras (status, expira_resgate_em);

COMMENT ON TABLE voucher_compras IS 'Compra de voucher no app: após pagamento aprovado, codigo_resgate e QR usados no posto (frentista).';
