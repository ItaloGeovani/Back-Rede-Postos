-- Litros associados a compra por campanha "por litro" (referência exibida no app).
ALTER TABLE voucher_compras
  ADD COLUMN IF NOT EXISTS litros NUMERIC(10, 3);

COMMENT ON COLUMN voucher_compras.litros IS 'Quantidade de litros negociada (campanha LITRO); nulo fora desse contexto.';
