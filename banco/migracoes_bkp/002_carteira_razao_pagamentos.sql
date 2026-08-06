BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'tipo_transacao_carteira') THEN
    CREATE TYPE tipo_transacao_carteira AS ENUM (
      'RECARGA',
      'RESERVA_COMPRA',
      'CONFIRMACAO_COMPRA',
      'ESTORNO_COMPRA',
      'CASHBACK',
      'BONUS',
      'AJUSTE'
    );
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'status_pagamento') THEN
    CREATE TYPE status_pagamento AS ENUM (
      'PENDENTE',
      'AUTORIZADO',
      'CAPTURADO',
      'FALHOU',
      'ESTORNADO',
      'CANCELADO'
    );
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS carteiras (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  usuario_id UUID NOT NULL REFERENCES usuarios(id),
  codigo_moeda TEXT NOT NULL DEFAULT 'BRL',
  nome_token TEXT NOT NULL,
  cotacao_token NUMERIC(18, 6) NOT NULL CHECK (cotacao_token > 0),
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rede_id, usuario_id)
);

CREATE TABLE IF NOT EXISTS transacoes_carteira (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  carteira_id UUID NOT NULL REFERENCES carteiras(id),
  tipo tipo_transacao_carteira NOT NULL,
  valor_fiat NUMERIC(18, 2) NOT NULL DEFAULT 0,
  valor_token NUMERIC(18, 6) NOT NULL,
  direcao SMALLINT NOT NULL CHECK (direcao IN (-1, 1)),
  tipo_referencia TEXT NOT NULL,
  referencia_id UUID NOT NULL,
  metadados JSONB NOT NULL DEFAULT '{}'::JSONB,
  ocorrido_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rede_id, tipo_referencia, referencia_id, tipo)
);

CREATE TABLE IF NOT EXISTS pagamentos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  usuario_id UUID NOT NULL REFERENCES usuarios(id),
  carteira_id UUID NOT NULL REFERENCES carteiras(id),
  nome_gateway TEXT NOT NULL,
  transacao_gateway_id TEXT NOT NULL,
  chave_idempotencia TEXT NOT NULL,
  valor_fiat NUMERIC(18, 2) NOT NULL CHECK (valor_fiat > 0),
  valor_token NUMERIC(18, 6) NOT NULL CHECK (valor_token > 0),
  status status_pagamento NOT NULL DEFAULT 'PENDENTE',
  autorizado_em TIMESTAMPTZ,
  capturado_em TIMESTAMPTZ,
  motivo_falha TEXT,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rede_id, nome_gateway, transacao_gateway_id),
  UNIQUE (rede_id, chave_idempotencia)
);

CREATE TABLE IF NOT EXISTS eventos_webhook_pagamento (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  nome_gateway TEXT NOT NULL,
  evento_gateway_id TEXT NOT NULL,
  tipo_evento TEXT NOT NULL,
  payload JSONB NOT NULL,
  recebido_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processado_em TIMESTAMPTZ,
  erro_processamento TEXT,
  UNIQUE (rede_id, nome_gateway, evento_gateway_id)
);

CREATE INDEX IF NOT EXISTS idx_carteiras_rede_usuario ON carteiras(rede_id, usuario_id);
CREATE INDEX IF NOT EXISTS idx_transacoes_carteira_rede_carteira_tempo ON transacoes_carteira(rede_id, carteira_id, ocorrido_em DESC);
CREATE INDEX IF NOT EXISTS idx_pagamentos_rede_status ON pagamentos(rede_id, status, criado_em DESC);
CREATE INDEX IF NOT EXISTS idx_eventos_webhook_pagamento_rede_recebido ON eventos_webhook_pagamento(rede_id, recebido_em DESC);

ALTER TABLE carteiras ENABLE ROW LEVEL SECURITY;
ALTER TABLE transacoes_carteira ENABLE ROW LEVEL SECURITY;
ALTER TABLE pagamentos ENABLE ROW LEVEL SECURITY;
ALTER TABLE eventos_webhook_pagamento ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS carteiras_isolamento_rede ON carteiras;
CREATE POLICY carteiras_isolamento_rede ON carteiras
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

DROP POLICY IF EXISTS transacoes_carteira_isolamento_rede ON transacoes_carteira;
CREATE POLICY transacoes_carteira_isolamento_rede ON transacoes_carteira
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

DROP POLICY IF EXISTS pagamentos_isolamento_rede ON pagamentos;
CREATE POLICY pagamentos_isolamento_rede ON pagamentos
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

DROP POLICY IF EXISTS eventos_webhook_pagamento_isolamento_rede ON eventos_webhook_pagamento;
CREATE POLICY eventos_webhook_pagamento_isolamento_rede ON eventos_webhook_pagamento
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

CREATE OR REPLACE VIEW saldos_carteira AS
SELECT
  tc.rede_id,
  tc.carteira_id,
  SUM(tc.valor_token * tc.direcao)::NUMERIC(18, 6) AS saldo_token,
  SUM(tc.valor_fiat * tc.direcao)::NUMERIC(18, 2) AS saldo_fiat_referencia
FROM transacoes_carteira tc
GROUP BY tc.rede_id, tc.carteira_id;

COMMIT;
