-- Niveis de cliente (Bronze, Prata, …): multiplicador de moeda (cashback, check-in, gire, etc.) e opcional de desconto.
BEGIN;

CREATE TABLE IF NOT EXISTS rede_niveis_cliente_config (
  rede_id UUID PRIMARY KEY REFERENCES redes (id) ON DELETE CASCADE,
  ativo BOOLEAN NOT NULL DEFAULT FALSE,
  mult_desconto_ativo BOOLEAN NOT NULL DEFAULT FALSE,
  niveis JSONB NOT NULL DEFAULT '[]'::jsonb,
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE rede_niveis_cliente_config DISABLE ROW LEVEL SECURITY;

COMMENT ON TABLE rede_niveis_cliente_config IS
  'Por rede: se ativo, ganhos de moeda (cashback, checkin, gire) usam mult por nivel; mult_desconto_ativo aplica mult em precos/descontos.';
COMMENT ON COLUMN rede_niveis_cliente_config.niveis IS
  'JSON array: {codigo, nome, mult_moeda, mult_desconto, ordem} — ver defaults no servidor.';

COMMIT;
