ALTER TABLE articles
ADD COLUMN scrapped_summary TEXT;

UPDATE articles
SET scrapped_summary = NULL;
