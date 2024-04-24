-- SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
--
-- SPDX-License-Identifier: AGPL-3.0-only

ALTER TABLE "bookmark" ADD COLUMN initial_url text NOT NULL DEFAULT '';
CREATE INDEX bookmark_url_idx ON "bookmark" (url);
CREATE INDEX bookmark_initial_url_idx ON "bookmark" (initial_url);

UPDATE "bookmark" SET initial_url = url;
