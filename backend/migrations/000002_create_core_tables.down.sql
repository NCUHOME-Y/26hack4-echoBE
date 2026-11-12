/*down文件与up文件严格反序*/
DROP TABLE IF EXISTS exp_entries;
ALTER TABLE users DROP COLUMN IF EXISTS exp;
DROP TABLE IF EXISTS replies;
DROP TABLE IF EXISTS question_skips;
DROP TABLE IF EXISTS votes;
ALTER TABLE questions DROP COLUMN IF EXISTS final_option_id;
DROP TABLE IF EXISTS options;
DROP TABLE IF EXISTS questions;
