BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'status_assinatura_rede') THEN
    CREATE TYPE status_assinatura_rede AS ENUM ('TESTE', 'ATIVA', 'EM_ATRASO', 'SUSPENSA', 'CANCELADA');
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS planos_cobranca (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  codigo TEXT NOT NULL UNIQUE,
  nome_exibicao TEXT NOT NULL,
  preco_mensal NUMERIC(18, 2) NOT NULL CHECK (preco_mensal >= 0),
  limite_postos INT NOT NULL CHECK (limite_postos > 0),
  limite_frentistas INT NOT NULL CHECK (limite_frentistas > 0),
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS assinaturas_rede (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL UNIQUE REFERENCES redes(id),
  plano_cobranca_id UUID NOT NULL REFERENCES planos_cobranca(id),
  status status_assinatura_rede NOT NULL DEFAULT 'TESTE',
  periodo_inicio TIMESTAMPTZ NOT NULL,
  periodo_fim TIMESTAMPTZ NOT NULL,
  prazo_tolerancia TIMESTAMPTZ,
  assinatura_externa_id TEXT,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS faturas_cobranca (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  assinatura_rede_id UUID NOT NULL REFERENCES assinaturas_rede(id),
  periodo_inicio TIMESTAMPTZ NOT NULL,
  periodo_fim TIMESTAMPTZ NOT NULL,
  valor_devido NUMERIC(18, 2) NOT NULL CHECK (valor_devido >= 0),
  valor_pago NUMERIC(18, 2) NOT NULL DEFAULT 0,
  vence_em TIMESTAMPTZ NOT NULL,
  pago_em TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'ABERTA',
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assinaturas_rede_status ON assinaturas_rede(status, periodo_fim);
CREATE INDEX IF NOT EXISTS idx_faturas_cobranca_rede_status ON faturas_cobranca(rede_id, status, vence_em);

ALTER TABLE assinaturas_rede ENABLE ROW LEVEL SECURITY;
ALTER TABLE faturas_cobranca ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS assinaturas_rede_isolamento_rede ON assinaturas_rede;
CREATE POLICY assinaturas_rede_isolamento_rede ON assinaturas_rede
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

DROP POLICY IF EXISTS faturas_cobranca_isolamento_rede ON faturas_cobranca;
CREATE POLICY faturas_cobranca_isolamento_rede ON faturas_cobranca
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

COMMIT;
