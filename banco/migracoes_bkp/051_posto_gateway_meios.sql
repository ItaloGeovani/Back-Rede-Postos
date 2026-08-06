-- Meios de pagamento por posto (intersectam com os da rede na compra).

ALTER TABLE postos
  ADD COLUMN IF NOT EXISTS gateway_meios_habilitados JSONB;

-- Backfill: copia os meios da rede; se nulo, padrão PIX.
UPDATE postos p
SET gateway_meios_habilitados = COALESCE(
  r.gateway_meios_habilitados,
  '{"pix": true, "cartao_credito": false, "cartao_debito": false, "dinheiro": false}'::jsonb
)
FROM redes r
WHERE p.rede_id = r.id
  AND p.gateway_meios_habilitados IS NULL;

ALTER TABLE postos
  ALTER COLUMN gateway_meios_habilitados SET DEFAULT '{"pix": true, "cartao_credito": false, "cartao_debito": false, "dinheiro": false}'::jsonb;

UPDATE postos
SET gateway_meios_habilitados = '{"pix": true, "cartao_credito": false, "cartao_debito": false, "dinheiro": false}'::jsonb
WHERE gateway_meios_habilitados IS NULL;

ALTER TABLE postos
  ALTER COLUMN gateway_meios_habilitados SET NOT NULL;

COMMENT ON COLUMN postos.gateway_meios_habilitados IS
  'Meios aceitos neste posto. Na compra: AND com redes.gateway_meios_habilitados.';
