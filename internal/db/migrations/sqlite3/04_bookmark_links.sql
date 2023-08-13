-- SPDX-FileCopyrightText: © 2023 Olivier Meunier <olivier@neokraft.net>
--
-- SPDX-License-Identifier: AGPL-3.0-only

ALTER TABLE "bookmark" ADD COLUMN links json NOT NULL DEFAULT "";
