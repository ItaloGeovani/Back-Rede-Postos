BEGIN;

CREATE TABLE IF NOT EXISTS rede_eventos_operacionais (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id) ON DELETE CASCADE,
  posto_id UUID REFERENCES postos(id) ON DELETE SET NULL,
  tipo_evento TEXT NOT NULL,
  entidade_tipo TEXT NOT NULL DEFAULT '',
  entidade_id UUID,
  titulo TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rede_eventos_operacionais_rede_criado
  ON rede_eventos_operacionais (rede_id, criado_em DESC);

CREATE INDEX IF NOT EXISTS idx_rede_eventos_operacionais_tipo
  ON rede_eventos_operacionais (tipo_evento);

CREATE TABLE IF NOT EXISTS rede_whatsapp_notificacoes (
  rede_id UUID PRIMARY KEY REFERENCES redes(id) ON DELETE CASCADE,
  habilitado BOOLEAN NOT NULL DEFAULT FALSE,
  instance_name TEXT NOT NULL DEFAULT '',
  instance_token TEXT NOT NULL DEFAULT '',
  group_jid TEXT NOT NULL DEFAULT '',
  notify_voucher_gerado BOOLEAN NOT NULL DEFAULT TRUE,
  notify_voucher_pago BOOLEAN NOT NULL DEFAULT TRUE,
  notify_voucher_baixa BOOLEAN NOT NULL DEFAULT TRUE,
  notify_campanha BOOLEAN NOT NULL DEFAULT TRUE,
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE rede_eventos_operacionais IS 'Logs operacionais da rede (voucher, campanha, etc.)';
COMMENT ON TABLE rede_whatsapp_notificacoes IS 'Credenciais Evolution Go por rede para aviso em grupo WhatsApp';

ALTER TABLE rede_eventos_operacionais DISABLE ROW LEVEL SECURITY;
ALTER TABLE rede_whatsapp_notificacoes DISABLE ROW LEVEL SECURITY;

COMMIT;
