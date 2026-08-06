-- Funcionalidades opcionais do app do cliente (por rede; padrao desligado).
BEGIN;

ALTER TABLE redes
  ADD COLUMN IF NOT EXISTS app_modulo_indique_ganhe BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS app_modulo_checkin_diario BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS app_modulo_gire_ganhe BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS app_modulo_redes_sociais BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN redes.app_modulo_indique_ganhe IS 'Exibir "Indique e ganhe" no app quando true.';
COMMENT ON COLUMN redes.app_modulo_checkin_diario IS 'Exibir "Check-in diario" no app quando true.';
COMMENT ON COLUMN redes.app_modulo_gire_ganhe IS 'Exibir "Gire e ganhe" no app quando true.';
COMMENT ON COLUMN redes.app_modulo_redes_sociais IS 'Exibir atalhos/area de redes sociais no app quando true.';

COMMIT;
