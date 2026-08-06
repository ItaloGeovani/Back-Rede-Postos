BEGIN;

-- Faixa de valor (R$) da compra para campanha com base VALOR_COMPRA; nulo = sem teto (legado).
ALTER TABLE campanhas
  ADD COLUMN IF NOT EXISTS valor_maximo_compra NUMERIC(14, 2);

COMMENT ON COLUMN campanhas.valor_maximo_compra IS
  'Teto de valor (R$) do voucher na base VALOR_COMPRA; NULL = sem teto; ignorado em LITRO.';

COMMIT;
