BEGIN;

CREATE TABLE IF NOT EXISTS premio_resgates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id) ON DELETE CASCADE,
  premio_id UUID NOT NULL REFERENCES premios(id) ON DELETE RESTRICT,
  usuario_id UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
  titulo_snapshot TEXT NOT NULL,
  imagem_url_snapshot TEXT,
  valor_moeda NUMERIC(18, 4) NOT NULL CHECK (valor_moeda > 0),
  status TEXT NOT NULL DEFAULT 'AGUARDANDO_RETIRADA'
    CHECK (status IN ('AGUARDANDO_RETIRADA', 'ENTREGUE', 'CANCELADO')),
  prazo_retirada_em TIMESTAMPTZ NOT NULL,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  entregue_em TIMESTAMPTZ,
  cancelado_em TIMESTAMPTZ,
  posto_entrega_id UUID REFERENCES postos(id) ON DELETE SET NULL,
  operador_usuario_id UUID REFERENCES usuarios(id) ON DELETE SET NULL,
  operador_papel TEXT,
  operador_nome_snapshot TEXT,
  motivo_cancelamento TEXT
);

CREATE INDEX IF NOT EXISTS idx_premio_resgates_rede_status_criado
  ON premio_resgates (rede_id, status, criado_em DESC);

CREATE INDEX IF NOT EXISTS idx_premio_resgates_usuario_criado
  ON premio_resgates (usuario_id, criado_em DESC);

CREATE INDEX IF NOT EXISTS idx_premio_resgates_premio
  ON premio_resgates (premio_id);

COMMENT ON TABLE premio_resgates IS 'Resgates de premios do catalogo com Luceninhas; retirada em qualquer posto da rede.';
COMMENT ON COLUMN premio_resgates.prazo_retirada_em IS 'Aviso: 2 dias uteis apos o resgate; nao expira automaticamente.';

ALTER TABLE premio_resgates DISABLE ROW LEVEL SECURITY;

COMMIT;
