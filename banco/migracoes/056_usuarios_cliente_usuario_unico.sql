-- Usuário de login do cliente (nome_completo) único por rede.
-- Novos cadastros usam só a-z0-9; login legado por e-mail continua.
-- Antes do índice: desambigua nomes duplicados já existentes (mesmo lower(trim) na rede).

WITH ranked AS (
  SELECT
    id,
    ROW_NUMBER() OVER (
      PARTITION BY rede_id, LOWER(TRIM(nome_completo))
      ORDER BY id
    ) AS rn
  FROM usuarios
  WHERE papel = 'cliente'::papel_usuario
    AND TRIM(COALESCE(nome_completo, '')) <> ''
)
UPDATE usuarios u
SET
  nome_completo = TRIM(u.nome_completo) || '_' || SUBSTRING(REPLACE(u.id::text, '-', ''), 1, 8),
  atualizado_em = NOW()
FROM ranked r
WHERE u.id = r.id
  AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS usuarios_rede_cliente_usuario_unico
  ON usuarios (rede_id, LOWER(TRIM(nome_completo)))
  WHERE papel = 'cliente'::papel_usuario
    AND TRIM(nome_completo) <> '';
