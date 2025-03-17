-- SPDX-FileCopyrightText: © 2025 Olivier Meunier <olivier@neokraft.net>
--
-- SPDX-License-Identifier: AGPL-3.0-only

ALTER TABLE token ADD COLUMN last_used datetime NULL;
ALTER TABLE credential ADD COLUMN last_used datetime NULL;
