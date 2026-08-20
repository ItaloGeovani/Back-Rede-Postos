-- Combustíveis passam a ser por posto (preço e disponibilidade individuais).
-- Copia cada combustível legado (sem posto) para todos os postos da mesma rede.

BEGIN;

ALTER TABLE rede_combustiveis
  ADD COLUMN IF NOT EXISTS posto_id UUID REFERENCES postos (id) ON DELETE CASCADE;

DROP INDEX IF EXISTS uq_rede_combustivel_codigo;

CREATE TEMP TABLE tmp_comb_map (
  old_id UUID NOT NULL,
  new_id UUID NOT NULL,
  posto_id UUID NOT NULL
) ON COMMIT DROP;

DO $$
DECLARE
  r RECORD;
  p RECORD;
  nid UUID;
BEGIN
  FOR r IN
    SELECT id, rede_id, nome, codigo, descricao, preco_por_litro, ativo, ordem
    FROM rede_combustiveis
    WHERE posto_id IS NULL
  LOOP
    FOR p IN
      SELECT id FROM postos WHERE rede_id = r.rede_id
    LOOP
      nid := gen_random_uuid();
      INSERT INTO rede_combustiveis (
        id, rede_id, posto_id, nome, codigo, descricao, preco_por_litro, ativo, ordem
      ) VALUES (
        nid, r.rede_id, p.id, r.nome, r.codigo, r.descricao, r.preco_por_litro, r.ativo, r.ordem
      );
      INSERT INTO tmp_comb_map (old_id, new_id, posto_id)
      VALUES (r.id, nid, p.id);
    END LOOP;
  END LOOP;
END $$;

-- Campanhas: cada vínculo antigo vira vínculo com todas as cópias por posto.
INSERT INTO campanha_combustiveis_rede (campanha_id, combustivel_rede_id)
SELECT ccr.campanha_id, m.new_id
FROM campanha_combustiveis_rede ccr
INNER JOIN tmp_comb_map m ON m.old_id = ccr.combustivel_rede_id
ON CONFLICT DO NOTHING;

DELETE FROM campanha_combustiveis_rede
WHERE combustivel_rede_id IN (SELECT DISTINCT old_id FROM tmp_comb_map);

-- Vouchers: preferir cópia do posto da compra; senão qualquer cópia.
UPDATE voucher_compras v
SET combustivel_rede_id = sub.new_id
FROM (
  SELECT DISTINCT ON (m.old_id, v2.id)
    v2.id AS compra_id,
    m.new_id
  FROM voucher_compras v2
  INNER JOIN tmp_comb_map m ON m.old_id = v2.combustivel_rede_id
  ORDER BY
    m.old_id,
    v2.id,
    CASE
      WHEN v2.posto_id_compra IS NOT NULL AND m.posto_id = v2.posto_id_compra THEN 0
      ELSE 1
    END,
    m.new_id
) sub
WHERE v.id = sub.compra_id;

-- Combustíveis sem posto e sem postos na rede: limpa vínculos e remove.
DELETE FROM campanha_combustiveis_rede
WHERE combustivel_rede_id IN (
  SELECT c.id
  FROM rede_combustiveis c
  WHERE c.posto_id IS NULL
    AND NOT EXISTS (SELECT 1 FROM postos p WHERE p.rede_id = c.rede_id)
);

UPDATE voucher_compras
SET combustivel_rede_id = NULL
WHERE combustivel_rede_id IN (
  SELECT c.id
  FROM rede_combustiveis c
  WHERE c.posto_id IS NULL
    AND NOT EXISTS (SELECT 1 FROM postos p WHERE p.rede_id = c.rede_id)
);

DELETE FROM rede_combustiveis WHERE posto_id IS NULL;

ALTER TABLE rede_combustiveis
  ALTER COLUMN posto_id SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_rede_combustivel_posto_codigo
  ON rede_combustiveis (posto_id, lower(trim(codigo)))
  WHERE codigo IS NOT NULL AND TRIM(codigo) <> '';

CREATE INDEX IF NOT EXISTS idx_rede_combustiveis_rede_posto
  ON rede_combustiveis (rede_id, posto_id);

CREATE INDEX IF NOT EXISTS idx_rede_combustiveis_posto_ordem
  ON rede_combustiveis (posto_id, ordem, nome);

COMMENT ON TABLE rede_combustiveis IS
  'Combustíveis por posto: nome, código e preço por litro do posto (gestor/gerente).';
COMMENT ON COLUMN rede_combustiveis.posto_id IS
  'Posto ao qual este combustível e preço pertencem.';

COMMIT;
