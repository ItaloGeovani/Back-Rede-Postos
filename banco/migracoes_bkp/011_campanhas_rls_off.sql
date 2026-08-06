BEGIN;

-- A API admin usa pool sem variaveis de sessao RLS; INSERT/SELECT em campanhas falhavam com politica restritiva.
ALTER TABLE campanhas DISABLE ROW LEVEL SECURITY;

COMMIT;
