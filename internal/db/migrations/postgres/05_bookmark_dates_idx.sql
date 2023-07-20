CREATE INDEX bookmark_created_idx ON "bookmark" USING btree (created DESC);
CREATE INDEX bookmark_updated_idx ON "bookmark" USING btree (updated DESC);
