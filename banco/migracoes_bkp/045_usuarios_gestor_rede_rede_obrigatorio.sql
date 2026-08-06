-- Gestor de rede deve estar sempre vinculado a uma rede (super_admin permanece com rede_id NULL).
-- Antes de aplicar: SELECT id, email FROM usuarios WHERE papel = 'gestor_rede' AND rede_id IS NULL;

BEGIN;

ALTER TABLE usuarios
ADD CONSTRAINT chk_usuarios_gestor_rede_rede_id_obrigatorio
CHECK (papel <> 'gestor_rede'::papel_usuario OR rede_id IS NOT NULL);

COMMIT;
