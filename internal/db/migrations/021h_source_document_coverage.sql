-- Reading coverage is document-scoped. Preserve the authoritative passage
-- relationship while avoiding thousands of unrelated passage-ID index probes.
ALTER TABLE passages ADD CONSTRAINT passages_org_id_id_document_id_key UNIQUE(org_id,id,document_id);
ALTER TABLE passage_sources ADD COLUMN document_id uuid;
UPDATE passage_sources s SET document_id=p.document_id FROM passages p WHERE p.org_id=s.org_id AND p.id=s.passage_id;
ALTER TABLE passage_sources ALTER COLUMN document_id SET NOT NULL;
ALTER TABLE passage_sources DROP CONSTRAINT passage_sources_org_id_passage_id_fkey;
ALTER TABLE passage_sources ADD CONSTRAINT passage_sources_org_id_passage_id_fkey
 FOREIGN KEY(org_id,passage_id,document_id) REFERENCES passages(org_id,id,document_id) ON DELETE CASCADE;
CREATE FUNCTION fill_passage_source_document() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.document_id IS NULL THEN
  SELECT document_id INTO NEW.document_id FROM passages WHERE org_id=NEW.org_id AND id=NEW.passage_id;
  IF NOT FOUND THEN
   RAISE EXCEPTION 'passage source parent missing' USING ERRCODE='foreign_key_violation',CONSTRAINT='passage_sources_org_id_passage_id_fkey';
  END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER passage_source_document BEFORE INSERT OR UPDATE OF org_id,passage_id,document_id ON passage_sources
 FOR EACH ROW EXECUTE FUNCTION fill_passage_source_document();
CREATE INDEX passage_sources_document_coverage_idx ON passage_sources(org_id,document_id) INCLUDE(outcome) WHERE outcome<>'pending';
