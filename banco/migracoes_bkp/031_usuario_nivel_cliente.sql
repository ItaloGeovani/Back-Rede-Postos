-- Nivel do cliente (bronze, prata, …) — usado com rede_niveis_cliente_config para multiplicar ganhos.
BEGIN;

ALTER TABLE usuarios
  ADD COLUMN IF NOT EXISTS nivel_cliente TEXT NOT NULL DEFAULT 'bronze';

COMMENT ON COLUMN usuarios.nivel_cliente IS
  'Codigo do nivel (ex. bronze) alinhado a rede_niveis_cliente; padrao bronze para novos clientes.';

CREATE INDEX IF NOT EXISTS idx_usuarios_rede_nivel_cliente
  ON usuarios (rede_id, nivel_cliente)
  WHERE papel = 'cliente'::papel_usuario;

COMMIT;
