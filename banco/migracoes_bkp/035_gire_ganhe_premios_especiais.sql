-- Prêmios especiais (jackpot): vários itens com valor e probabilidade; sorteio no servidor.

ALTER TABLE IF EXISTS rede_gire_ganhe_config
  ADD COLUMN IF NOT EXISTS premios_especiais_ativo BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS premios_especiais JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE IF EXISTS gire_ganhe_giros
  ADD COLUMN IF NOT EXISTS premio_especial BOOLEAN NOT NULL DEFAULT FALSE;
