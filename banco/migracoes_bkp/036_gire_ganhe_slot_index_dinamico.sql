-- Gire e ganhe: slot_index passa a suportar quantidade dinamica de setores.

ALTER TABLE IF EXISTS gire_ganhe_giros
  DROP CONSTRAINT IF EXISTS gire_ganhe_giros_slot_index_check;

ALTER TABLE IF EXISTS gire_ganhe_giros
  ADD CONSTRAINT gire_ganhe_giros_slot_index_check
  CHECK (slot_index >= 0 AND slot_index < 100);
