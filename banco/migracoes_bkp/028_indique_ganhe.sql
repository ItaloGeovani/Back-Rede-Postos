-- Indique e ganhe: regra, premios em moeda da rede, codigo por usuario, vinculo indicado->referente.
BEGIN;

CREATE TABLE IF NOT EXISTS rede_indique_ganhe_config (
  rede_id UUID PRIMARY KEY REFERENCES redes (id) ON DELETE CASCADE,
  regra TEXT NOT NULL DEFAULT 'PRIMEIRA_COMPRA_VOUCHER' CHECK (regra IN ('CADASTRAR', 'PRIMEIRA_COMPRA_VOUCHER')),
  moedas_premio_referente NUMERIC(18, 6) NOT NULL DEFAULT 0 CHECK (moedas_premio_referente >= 0),
  moedas_premio_indicado NUMERIC(18, 6) NOT NULL DEFAULT 0 CHECK (moedas_premio_indicado >= 0),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE rede_indique_ganhe_config DISABLE ROW LEVEL SECURITY;

ALTER TABLE usuarios
  ADD COLUMN IF NOT EXISTS codigo_indicacao TEXT;

-- Codigo exibido no app: unico por rede (case-insensitive).
CREATE UNIQUE INDEX IF NOT EXISTS uq_usuarios_rede_codigo_indicacao
  ON usuarios (rede_id, lower(trim(codigo_indicacao)))
  WHERE codigo_indicacao IS NOT NULL AND trim(codigo_indicacao) <> '';

CREATE TABLE IF NOT EXISTS indicacoes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes (id) ON DELETE CASCADE,
  referente_usuario_id UUID NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
  indicado_usuario_id UUID NOT NULL REFERENCES usuarios (id) ON DELETE CASCADE,
  codigo_informado TEXT,
  premiado_cadastro_referente BOOLEAN NOT NULL DEFAULT FALSE,
  premiado_cadastro_indicado BOOLEAN NOT NULL DEFAULT FALSE,
  premiado_compra_referente BOOLEAN NOT NULL DEFAULT FALSE,
  premiado_compra_indicado BOOLEAN NOT NULL DEFAULT FALSE,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_indicacoes_indicado UNIQUE (indicado_usuario_id),
  CONSTRAINT ck_indicacoes_distintos CHECK (referente_usuario_id <> indicado_usuario_id)
);

CREATE INDEX IF NOT EXISTS idx_indicacoes_rede_referente
  ON indicacoes (rede_id, referente_usuario_id);

CREATE INDEX IF NOT EXISTS idx_indicacoes_rede_indicado
  ON indicacoes (rede_id, indicado_usuario_id);

ALTER TABLE indicacoes DISABLE ROW LEVEL SECURITY;

COMMENT ON TABLE rede_indique_ganhe_config IS 'Regra e premios em unidades de moeda virtual; modulos liga no painel (redes.app_modulo_*)';
COMMENT ON TABLE indicacoes IS 'Indicado unico: cada cliente so pode ser vinculado a um referente.';

COMMIT;
