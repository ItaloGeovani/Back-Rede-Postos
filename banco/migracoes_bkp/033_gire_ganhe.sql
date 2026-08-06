-- Gire e ganhe (roleta): configuracao por rede e historico de giros por cliente.

CREATE TABLE IF NOT EXISTS rede_gire_ganhe_config (
  rede_id UUID NOT NULL PRIMARY KEY REFERENCES redes(id) ON DELETE CASCADE,
  custo_moedas NUMERIC(18, 4) NOT NULL DEFAULT 10 CHECK (custo_moedas > 0),
  premio_min_moedas NUMERIC(18, 4) NOT NULL DEFAULT 1 CHECK (premio_min_moedas >= 0),
  premio_max_moedas NUMERIC(18, 4) NOT NULL DEFAULT 20 CHECK (premio_max_moedas >= premio_min_moedas),
  giros_max_dia INTEGER NOT NULL DEFAULT 1 CHECK (giros_max_dia >= 1 AND giros_max_dia <= 100),
  timezone TEXT NOT NULL DEFAULT 'America/Sao_Paulo',
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gire_ganhe_giros (
  id UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id) ON DELETE CASCADE,
  usuario_id UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
  ciclo_dia DATE NOT NULL,
  numero_no_dia INTEGER NOT NULL CHECK (numero_no_dia >= 1),
  slot_index INTEGER NOT NULL CHECK (slot_index >= 0 AND slot_index < 12),
  premio_base_moedas NUMERIC(18, 6) NOT NULL CHECK (premio_base_moedas >= 0),
  premio_creditado_moedas NUMERIC(18, 6) NOT NULL CHECK (premio_creditado_moedas >= 0),
  multiplicador_nivel NUMERIC(12, 6) NOT NULL DEFAULT 1 CHECK (multiplicador_nivel > 0),
  custo_debitado_moedas NUMERIC(18, 6) NOT NULL DEFAULT 0 CHECK (custo_debitado_moedas >= 0),
  giro_gratis BOOLEAN NOT NULL DEFAULT FALSE,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gire_giros_rede_usuario_ciclo
  ON gire_ganhe_giros (rede_id, usuario_id, ciclo_dia DESC, criado_em DESC);

CREATE INDEX IF NOT EXISTS idx_gire_giros_rede_usuario_total
  ON gire_ganhe_giros (rede_id, usuario_id, criado_em DESC);
