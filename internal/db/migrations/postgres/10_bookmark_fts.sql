-- SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
--
-- SPDX-License-Identifier: AGPL-3.0-only

-- New search configuration
DROP TEXT search CONFIGURATION ts;

CREATE TEXT SEARCH CONFIGURATION ts (COPY = simple);

ALTER TEXT SEARCH CONFIGURATION ts
ALTER MAPPING FOR hword, hword_part, word, host
WITH unaccent, english_stem, french_stem;

-- Index everything again
DELETE FROM bookmark_search;

DROP INDEX IF EXISTS bookmark_search_text_idx;
CREATE INDEX bookmark_search_all_idx ON bookmark_search USING GIN((title || description || "text" || site || "label"));

INSERT INTO bookmark_search (
    bookmark_id, title, description, "text", site, author, "label"
) SELECT
    id,
    setweight(to_tsvector('ts', title), 'A'),
    to_tsvector('ts', description),
    to_tsvector('ts', "text"),
    to_tsvector('ts',
        site_name || ' ' || domain || ' ' ||
        REGEXP_REPLACE(site, '^www\.', '') || ' ' ||
        REPLACE(domain, '.', ' ') ||
        REPLACE(REGEXP_REPLACE(site, '^www\.', ''), '.', ' ')
    ),
    jsonb_to_tsvector('ts', authors, '["string"]'),
    setweight(jsonb_to_tsvector('ts', labels, '["string"]'), 'A')
FROM bookmark;
