-- Corrige auditoria: nome/papel do operador na baixa deve refletir o cadastro em usuarios
-- (evita snapshot errado quando sessao_api guardou nome desatualizado ou trocado).
UPDATE voucher_compras v
SET
  operador_nome_snapshot = TRIM(u.nome_completo),
  operador_papel = TRIM(u.papel::text)
FROM usuarios u
WHERE v.operador_usuario_id = u.id
  AND u.rede_id = v.rede_id
  AND v.operador_usuario_id IS NOT NULL;
