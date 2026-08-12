BEGIN;

-- Código de acesso do frentista (login / baixa no PC compartilhado).
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS codigo_acesso TEXT NULL;

COMMENT ON COLUMN usuarios.codigo_acesso IS 'Código de login do frentista (único por posto); texto ou números.';

-- E-mail opcional para frentistas novos (login por código).
ALTER TABLE usuarios ALTER COLUMN email DROP NOT NULL;

-- Troca UNIQUE (rede_id, email) por índice parcial (ignora e-mail nulo/vazio).
ALTER TABLE usuarios DROP CONSTRAINT IF EXISTS usuarios_rede_id_email_key;

CREATE UNIQUE INDEX IF NOT EXISTS usuarios_rede_email_unico
  ON usuarios (rede_id, LOWER(TRIM(email)))
  WHERE email IS NOT NULL AND TRIM(email) <> '';

-- Código único por posto quando preenchido.
CREATE UNIQUE INDEX IF NOT EXISTS usuarios_posto_codigo_acesso_unico
  ON usuarios (posto_id, LOWER(TRIM(codigo_acesso)))
  WHERE posto_id IS NOT NULL
    AND codigo_acesso IS NOT NULL
    AND TRIM(codigo_acesso) <> '';

COMMIT;
