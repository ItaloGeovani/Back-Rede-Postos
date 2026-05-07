BEGIN;

ALTER TABLE campanhas
  ADD COLUMN IF NOT EXISTS tipo_beneficio TEXT NOT NULL DEFAULT 'DESCONTO'
    CHECK (tipo_beneficio IN ('DESCONTO', 'CASHBACK'));

UPDATE campanhas
SET tipo_beneficio = 'DESCONTO'
WHERE tipo_beneficio IS NULL OR trim(tipo_beneficio) = '';

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'chk_campanhas_cashback_percentual'
  ) THEN
    ALTER TABLE campanhas
      ADD CONSTRAINT chk_campanhas_cashback_percentual
      CHECK (
        tipo_beneficio <> 'CASHBACK'
        OR modalidade_desconto = 'PERCENTUAL'
      );
  END IF;
END $$;

COMMENT ON COLUMN campanhas.tipo_beneficio IS
  'DESCONTO reduz o valor final; CASHBACK credita moeda apos pagamento confirmado.';

COMMIT;
