-- Check-in diario: config por rede + registo por usuario/ciclo (janela apos hora local).

CREATE TABLE IF NOT EXISTS rede_checkin_diario_config (
  rede_id UUID NOT NULL PRIMARY KEY REFERENCES redes(id) ON DELETE CASCADE,
  moedas_por_dia NUMERIC(18, 4) NOT NULL DEFAULT 10 CHECK (moedas_por_dia > 0),
  hora_abertura TIME NOT NULL DEFAULT '12:00:00'::time,
  timezone TEXT NOT NULL DEFAULT 'America/Sao_Paulo',
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS checkin_diario_registos (
  id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id) ON DELETE CASCADE,
  usuario_id UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
  ciclo_key DATE NOT NULL,
  valor_moedas NUMERIC(18, 6) NOT NULL CHECK (valor_moedas > 0),
  streak INTEGER NOT NULL DEFAULT 1 CHECK (streak >= 1),
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rede_id, usuario_id, ciclo_key)
);

CREATE INDEX IF NOT EXISTS idx_checkin_reg_rede_usuario_ciclo
  ON checkin_diario_registos (rede_id, usuario_id, ciclo_key DESC);
