-- SPDX-FileCopyrightText: © 2024 Olivier Meunier <olivier@neokraft.net>
--
-- SPDX-License-Identifier: AGPL-3.0-only


ALTER TABLE "bookmark" ADD COLUMN read_progress smallint NOT NULL DEFAULT 0;
ALTER TABLE "bookmark" ADD COLUMN read_anchor   text NOT NULL DEFAULT '';

UPDATE bookmark SET read_progress = 100 WHERE is_archived = true;
