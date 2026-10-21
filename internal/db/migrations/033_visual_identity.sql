CREATE TABLE visual_identity (
 org_id uuid NOT NULL,
 document_id uuid NOT NULL,
 status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','ready','unavailable')),
 result jsonb NOT NULL DEFAULT '{}',
 elapsed_ms integer NOT NULL DEFAULT 0,
 PRIMARY KEY(org_id,document_id),
 FOREIGN KEY(org_id,document_id) REFERENCES documents(org_id,id) ON DELETE CASCADE,
 CHECK (octet_length(result::text) <= 4194304)
);
