BEGIN;

-- Uma moeda virtual por rede (nome exibido + quantas unidades por R$ 1,00 na conversao).
ALTER TABLE redes
  ADD COLUMN IF NOT EXISTS moeda_virtual_nome TEXT NOT NULL DEFAULT 'Creditos',
  ADD COLUMN IF NOT EXISTS moeda_virtual_cotacao NUMERIC(18, 6) NOT NULL DEFAULT 1
    CHECK (moeda_virtual_cotacao > 0);

COMMENT ON COLUMN redes.moeda_virtual_nome IS 'Nome da moeda (ex.: NioCoins) — unica por rede';
COMMENT ON COLUMN redes.moeda_virtual_cotacao IS 'Unidades de moeda por R$ 1,00 na recarga/conversao (ex.: 1 = R$1 compra 1 unidade)';

COMMIT;
