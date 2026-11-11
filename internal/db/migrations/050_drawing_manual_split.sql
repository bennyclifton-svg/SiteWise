-- A user can split a drawing set the automatic split stopped on. A manual
-- split files every page, keeping Jev's reading of pages it did not confirm
-- as drawings, instead of keeping the whole set as one document.
ALTER TABLE drawing_expansions ADD COLUMN manual boolean NOT NULL DEFAULT false;
