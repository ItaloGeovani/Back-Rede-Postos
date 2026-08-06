-- Gire e ganhe: opcao de primeiro giro gratis (uma vez na vida).

ALTER TABLE IF EXISTS rede_gire_ganhe_config
  ADD COLUMN IF NOT EXISTS primeiro_giro_gratis_ativo BOOLEAN NOT NULL DEFAULT TRUE;
