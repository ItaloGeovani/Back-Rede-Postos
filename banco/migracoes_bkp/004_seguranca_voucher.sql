BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'status_voucher') THEN
    CREATE TYPE status_voucher AS ENUM (
      'PENDENTE',
      'USADO',
      'EXPIRADO',
      'CANCELADO',
      'REJEITADO'
    );
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS vouchers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  posto_id UUID REFERENCES postos(id),
  usuario_id UUID NOT NULL REFERENCES usuarios(id),
  pagamento_id UUID REFERENCES pagamentos(id),
  tipo_combustivel TEXT NOT NULL,
  litros NUMERIC(10, 3) NOT NULL CHECK (litros > 0),
  custo_token NUMERIC(18, 6) NOT NULL CHECK (custo_token > 0),
  nonce UUID NOT NULL DEFAULT gen_random_uuid(),
  hash_payload TEXT NOT NULL,
  assinatura_hmac TEXT NOT NULL,
  expira_em TIMESTAMPTZ NOT NULL,
  status status_voucher NOT NULL DEFAULT 'PENDENTE',
  validado_em TIMESTAMPTZ,
  validado_por UUID REFERENCES usuarios(id),
  motivo_rejeicao TEXT,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rede_id, nonce)
);

CREATE TABLE IF NOT EXISTS tentativas_validacao_voucher (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  voucher_id UUID REFERENCES vouchers(id),
  nonce UUID,
  usuario_frentista_id UUID REFERENCES usuarios(id),
  posto_id UUID REFERENCES postos(id),
  resultado TEXT NOT NULL,
  motivo TEXT,
  payload_requisicao JSONB,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vouchers_rede_status_expira ON vouchers(rede_id, status, expira_em);
CREATE INDEX IF NOT EXISTS idx_vouchers_rede_usuario ON vouchers(rede_id, usuario_id, criado_em DESC);
CREATE INDEX IF NOT EXISTS idx_tentativas_validacao_voucher_rede_tempo ON tentativas_validacao_voucher(rede_id, criado_em DESC);

ALTER TABLE vouchers ENABLE ROW LEVEL SECURITY;
ALTER TABLE tentativas_validacao_voucher ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS vouchers_isolamento_rede ON vouchers;
CREATE POLICY vouchers_isolamento_rede ON vouchers
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

DROP POLICY IF EXISTS tentativas_validacao_voucher_isolamento_rede ON tentativas_validacao_voucher;
CREATE POLICY tentativas_validacao_voucher_isolamento_rede ON tentativas_validacao_voucher
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

COMMIT;
