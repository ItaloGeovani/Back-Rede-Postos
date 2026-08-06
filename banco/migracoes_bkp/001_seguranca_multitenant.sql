BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'papel_usuario') THEN
    CREATE TYPE papel_usuario AS ENUM (
      'super_admin',
      'gestor_rede',
      'gerente_posto',
      'frentista',
      'cliente'
    );
  END IF;
END $$;

CREATE TABLE IF NOT EXISTS redes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  razao_social TEXT NOT NULL,
  nome_fantasia TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ATIVA',
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS postos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID NOT NULL REFERENCES redes(id),
  nome TEXT NOT NULL,
  codigo TEXT NOT NULL,
  cidade TEXT,
  estado TEXT,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rede_id, codigo)
);

CREATE TABLE IF NOT EXISTS usuarios (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID REFERENCES redes(id),
  posto_id UUID REFERENCES postos(id),
  papel papel_usuario NOT NULL,
  nome_completo TEXT NOT NULL,
  email TEXT NOT NULL,
  senha_hash TEXT NOT NULL,
  ativo BOOLEAN NOT NULL DEFAULT TRUE,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  atualizado_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (rede_id, email)
);

CREATE TABLE IF NOT EXISTS logs_auditoria (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rede_id UUID,
  usuario_ator_id UUID,
  tipo_evento TEXT NOT NULL,
  tipo_entidade TEXT NOT NULL,
  entidade_id UUID,
  dados_anteriores JSONB,
  dados_novos JSONB,
  ip_origem INET,
  agente_usuario TEXT,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_postos_rede_id ON postos(rede_id, criado_em DESC);
CREATE INDEX IF NOT EXISTS idx_usuarios_rede_id ON usuarios(rede_id, criado_em DESC);
CREATE INDEX IF NOT EXISTS idx_logs_auditoria_rede_id ON logs_auditoria(rede_id, criado_em DESC);
CREATE INDEX IF NOT EXISTS idx_logs_auditoria_tipo_evento ON logs_auditoria(tipo_evento, criado_em DESC);

ALTER TABLE postos ENABLE ROW LEVEL SECURITY;
ALTER TABLE usuarios ENABLE ROW LEVEL SECURITY;
ALTER TABLE logs_auditoria ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS postos_isolamento_rede ON postos;
CREATE POLICY postos_isolamento_rede ON postos
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

DROP POLICY IF EXISTS usuarios_isolamento_rede ON usuarios;
CREATE POLICY usuarios_isolamento_rede ON usuarios
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

DROP POLICY IF EXISTS logs_auditoria_isolamento_rede ON logs_auditoria;
CREATE POLICY logs_auditoria_isolamento_rede ON logs_auditoria
USING (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
)
WITH CHECK (
  current_setting('app.papel_atual', true) = 'super_admin'
  OR rede_id = NULLIF(current_setting('app.rede_atual_id', true), '')::UUID
);

COMMIT;
