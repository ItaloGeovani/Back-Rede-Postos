BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'status_campanha') THEN
    CREATE TYPE status_campanha AS ENUM ('RASCUNHO', 'ATIVA', 'PAUSADA', 'ARQUIVADA');
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'tipo_regra_campanha') THEN
    CREATE TYPE tipo_regra_campanha AS ENUM ('CASHBACK', 'BONUS_VOLUME', 'PRECO_DINAMICO');
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS campanhas (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  nome TEXT NOT NULL,
  descricao TEXT,
  status status_campanha NOT NULL DEFAULT 'RASCUNHO',
  criado_por UUID NOT NULL REFERENCES usuarios(id),
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS versoes_campanha (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  campanha_id UUID NOT NULL REFERENCES campanhas(id) ON DELETE CASCADE,
  numero_versao INT NOT NULL CHECK (numero_versao > 0),
  inicia_em TIMESTAMPTZ NOT NULL,
  termina_em TIMESTAMPTZ,
  publicada BOOLEAN NOT NULL DEFAULT FALSE,
  publicada_em TIMESTAMPTZ,
  publicada_por UUID REFERENCES usuarios(id),
  checksum TEXT NOT NULL,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rede_id, campanha_id, numero_versao)
);

CREATE TABLE IF NOT EXISTS regras_campanha (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  campanha_id UUID NOT NULL REFERENCES campanhas(id) ON DELETE CASCADE,
  versao_campanha_id UUID NOT NULL REFERENCES versoes_campanha(id) ON DELETE CASCADE,
  tipo_regra tipo_regra_campanha NOT NULL,
  prioridade SMALLINT NOT NULL CHECK (prioridade BETWEEN 1 AND 100),
  parar_ao_casar BOOLEAN NOT NULL DEFAULT FALSE,
  condicoes JSONB NOT NULL,
  resultados JSONB NOT NULL,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS simulacoes_campanha (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  versao_campanha_id UUID NOT NULL REFERENCES versoes_campanha(id) ON DELETE CASCADE,
  payload_entrada JSONB NOT NULL,
  trilha_decisao JSONB NOT NULL,
  payload_saida JSONB NOT NULL,
  simulado_por UUID NOT NULL REFERENCES usuarios(id),
  simulado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campanhas_rede_status ON campanhas(rede_id, status, criado_em DESC);
CREATE INDEX IF NOT EXISTS idx_versoes_campanha_rede_tempo ON versoes_campanha(rede_id, inicia_em, termina_em);
CREATE INDEX IF NOT EXISTS idx_regras_campanha_rede_prioridade ON regras_campanha(rede_id, versao_campanha_id, prioridade);
CREATE INDEX IF NOT EXISTS idx_simulacoes_campanha_rede_tempo ON simulacoes_campanha(rede_id, simulado_em DESC);

ALTER TABLE campanhas ENABLE ROW LEVEL SECURITY;
ALTER TABLE versoes_campanha ENABLE ROW LEVEL SECURITY;
ALTER TABLE regras_campanha ENABLE ROW LEVEL SECURITY;
ALTER TABLE simulacoes_campanha ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS campanhas_isolamento_rede ON campanhas;
CREATE POLICY campanhas_isolamento_rede ON campanhas
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

DROP POLICY IF EXISTS versoes_campanha_isolamento_rede ON versoes_campanha;
CREATE POLICY versoes_campanha_isolamento_rede ON versoes_campanha
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

DROP POLICY IF EXISTS regras_campanha_isolamento_rede ON regras_campanha;
CREATE POLICY regras_campanha_isolamento_rede ON regras_campanha
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

DROP POLICY IF EXISTS simulacoes_campanha_isolamento_rede ON simulacoes_campanha;
CREATE POLICY simulacoes_campanha_isolamento_rede ON simulacoes_campanha
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

COMMIT;
