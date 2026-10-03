CREATE TABLE drawing_expansions (
    org_id uuid NOT NULL,
    source_id uuid NOT NULL,
    page_count integer NOT NULL CHECK (page_count > 1),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'complete', 'review')),
    reason text NOT NULL DEFAULT '',
    PRIMARY KEY (org_id, source_id),
    FOREIGN KEY (org_id, source_id) REFERENCES documents(org_id, id) ON DELETE CASCADE
);

CREATE TABLE drawing_sheets (
    org_id uuid NOT NULL,
    source_id uuid NOT NULL,
    page_number integer NOT NULL CHECK (page_number > 0),
    document_id uuid NOT NULL,
    PRIMARY KEY (org_id, source_id, page_number),
    UNIQUE (org_id, document_id),
    FOREIGN KEY (org_id, source_id) REFERENCES drawing_expansions(org_id, source_id) ON DELETE CASCADE,
    FOREIGN KEY (org_id, document_id) REFERENCES documents(org_id, id) ON DELETE CASCADE,
    CHECK (source_id <> document_id)
);
