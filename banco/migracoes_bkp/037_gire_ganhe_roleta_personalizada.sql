-- Roleta: modo padrão (min–max + fatias iguais + jackpots opcionais) ou personalizado (lista valor + % = 100%).

ALTER TABLE IF EXISTS rede_gire_ganhe_config
  ADD COLUMN IF NOT EXISTS roleta_modo TEXT NOT NULL DEFAULT 'padrao',
  ADD COLUMN IF NOT EXISTS premios_roleta_personalizada JSONB NOT NULL DEFAULT '[]'::jsonb;
