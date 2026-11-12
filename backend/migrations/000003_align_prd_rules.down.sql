/*回退 000003，顺序与 up 严格相反*/
DROP INDEX IF EXISTS questions_contested_idx;
ALTER TABLE questions DROP COLUMN IF EXISTS contested_at;

ALTER TABLE options DROP CONSTRAINT IF EXISTS options_sort_no_range;
ALTER TABLE questions DROP COLUMN IF EXISTS option_count;

ALTER TABLE users ALTER COLUMN nickname TYPE VARCHAR(64);
ALTER TABLE questions DROP CONSTRAINT questions_description_check;
ALTER TABLE questions ALTER COLUMN description TYPE VARCHAR(102);
ALTER TABLE questions ADD CONSTRAINT questions_description_check CHECK (char_length(description) BETWEEN 1 AND 102);