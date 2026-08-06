-- Corrige campanhas legadas: tipo CASHBACK gravado como DESCONTO e percentuais em fração (0.05 = 5%).
-- Após esta migração, tipo_beneficio no banco é a fonte da verdade (sem depender de texto no nome).

BEGIN;

-- 1) Percentual armazenado como fração (0 < valor < 1) → escala 0–100 (ex.: 0.05 → 5, 0.1 → 10)
UPDATE campanhas
SET
  valor_desconto = round((valor_desconto * 100)::numeric, 4),
  atualizado_em = NOW()
WHERE modalidade_desconto = 'PERCENTUAL'
  AND valor_desconto > 0
  AND valor_desconto < 1;

-- 2) Campanhas de cashback que ficaram como DESCONTO (heurística conservadora: texto + percentual sobre compra)
UPDATE campanhas
SET
  tipo_beneficio = 'CASHBACK',
  atualizado_em = NOW()
WHERE tipo_beneficio = 'DESCONTO'
  AND modalidade_desconto = 'PERCENTUAL'
  AND coalesce(nullif(trim(base_desconto), ''), 'VALOR_COMPRA') = 'VALOR_COMPRA'
  AND (
    lower(coalesce(nome, '')) LIKE '%cashback%'
    OR lower(coalesce(nome, '')) LIKE '%cash back%'
    OR lower(coalesce(titulo, '')) LIKE '%cashback%'
    OR lower(coalesce(titulo, '')) LIKE '%cash back%'
    OR lower(coalesce(descricao, '')) LIKE '%cashback%'
    OR lower(coalesce(descricao, '')) LIKE '%cash back%'
  );

COMMENT ON COLUMN campanhas.tipo_beneficio IS
  'DESCONTO: reduz valor_final do PIX na compra. CASHBACK: valor_final integral; credita moeda virtual apos pagamento (percentual obrigatorio). valor_desconto em PERCENTUAL: 0-100 (ex.: 10 = 10%).';

COMMIT;
