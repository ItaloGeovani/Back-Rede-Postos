-- Tokens FCM do app cliente (push); um utilizador pode ter vários dispositivos.
CREATE TABLE IF NOT EXISTS usuario_fcm_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  usuario_id UUID NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
  token TEXT NOT NULL,
  plataforma TEXT NOT NULL CHECK (plataforma IN ('android', 'ios', 'web')),
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_usuario_fcm_token UNIQUE (token)
);

CREATE INDEX IF NOT EXISTS idx_usuario_fcm_tokens_usuario_id ON usuario_fcm_tokens (usuario_id);

COMMENT ON TABLE usuario_fcm_tokens IS 'Registo de tokens Firebase Cloud Messaging por utilizador (app fechado / push).';
