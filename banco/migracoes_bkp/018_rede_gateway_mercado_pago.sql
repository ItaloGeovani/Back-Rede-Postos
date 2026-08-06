-- Credenciais Mercado Pago por rede (PIX / webhooks). Não usar .env global.
CREATE TABLE IF NOT EXISTS rede_mercado_pago (
  rede_id UUID PRIMARY KEY REFERENCES redes (id) ON DELETE CASCADE,
  mp_access_token TEXT,
  mp_webhook_secret TEXT,
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE rede_mercado_pago IS 'Mercado Pago: access token e secret de validação de webhook por rede.';
