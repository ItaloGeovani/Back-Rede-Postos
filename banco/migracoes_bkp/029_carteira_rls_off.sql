-- A API Go usa pool sem variáveis de sessão (app.rede_atual_id / app.papel_atual).
-- Com RLS ativo, INSERT em transacoes_carteira (bônus indique e ganhe, etc.) falha para o role da app.
-- Alinhado ao desligar RLS em campanhas (011_campanhas_rls_off.sql).
BEGIN;

ALTER TABLE IF EXISTS carteiras DISABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS transacoes_carteira DISABLE ROW LEVEL SECURITY;

COMMIT;
