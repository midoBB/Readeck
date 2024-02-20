-- SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
--
-- SPDX-License-Identifier: AGPL-3.0-only

DROP TABLE bookmark_idx;

CREATE VIRTUAL TABLE IF NOT EXISTS bookmark_idx USING fts5(
    tokenize='unicode61 remove_diacritics 2',
    content='bookmark',
    content_rowid='id',
    catchall,
    title,
    description,
    text,
    site,
    author,
    label
);

INSERT INTO bookmark_idx(bookmark_idx, rank) VALUES ('rank', 'bm25(0, 12.0, 6.0, 5.0, 2.0, 4.0)');

INSERT INTO bookmark_idx (rowid, catchall, title, description, text, site, author, label)
SELECT b.id, 'oooooo', b.title, b.description, b.text,
    b.site_name || ' ' || b.site || ' ' || b.domain,
    b.authors, b.labels
FROM bookmark b;

DROP TRIGGER IF EXISTS bookmark_ai;
CREATE TRIGGER bookmark_ai AFTER INSERT ON bookmark BEGIN
    INSERT INTO bookmark_idx (
        rowid, catchall, title, description, text, site, author, label
    ) VALUES (
        new.id, 'oooooo', new.title, new.description, new.text, new.site_name || ' ' || new.site || ' ' || new.domain, new.authors, new.labels
    );
END;

DROP TRIGGER IF EXISTS bookmark_au;
CREATE TRIGGER bookmark_au AFTER UPDATE ON bookmark BEGIN
    INSERT INTO bookmark_idx(
        bookmark_idx, rowid, catchall, title, description, text, site, author, label
    ) VALUES (
        'delete', old.id, 'oooooo', old.title, old.description, old.text, old.site, old.authors, old.labels
    );
    INSERT INTO bookmark_idx (
        rowid, catchall, title, description, text, site, author, label
    ) VALUES (
        new.id, 'oooooo', new.title, new.description, new.text, new.site_name || ' ' || new.site || ' ' || new.domain, new.authors, new.labels
    );
END;

DROP TRIGGER IF EXISTS bookmark_ad;
CREATE TRIGGER IF NOT EXISTS bookmark_ad AFTER DELETE ON bookmark BEGIN
    INSERT INTO bookmark_idx(
        bookmark_idx, rowid, catchall, title, description, text, site, author, label
    ) VALUES (
        'delete', old.id, 'oooooo', old.title, old.description, old.text, old.site, old.authors, old.labels
    );
END;
