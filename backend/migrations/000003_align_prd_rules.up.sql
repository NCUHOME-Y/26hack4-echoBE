
/*文案长度：问题描述 102 改 100，昵称 64 改 7*/
ALTER TABLE questions DROP CONSTRAINT questions_description_check; /*先删旧 CHECK，否则改类型时会被 102 的上限卡住*/
ALTER TABLE questions ALTER COLUMN description TYPE VARCHAR(100);
ALTER TABLE questions ADD CONSTRAINT questions_description_check CHECK (char_length(description) BETWEEN 1 AND 100);
ALTER TABLE users ALTER COLUMN nickname TYPE VARCHAR(7); /*中英文都按一个字符算*/

/*选项数量只能 2/3/4 条。跨行规则数据库表达不了，拆成两层拦*/
ALTER TABLE questions ADD COLUMN option_count SMALLINT NOT NULL DEFAULT 2 CHECK (option_count IN (2,3,4)); /*下限：发布时声明有几条，默认 2 兼容已有数据*/
ALTER TABLE options ADD CONSTRAINT options_sort_no_range CHECK (sort_no BETWEEN 1 AND 4); /*上限：第 5 条的 sort_no 必然是 5，直接拦死*/

/*争议改为结算后判定，用入库时间标记，不再靠 status='contested'*/
ALTER TABLE questions ADD COLUMN contested_at TIMESTAMPTZ; /*NULL 表示未进争议库*/
CREATE INDEX questions_contested_idx ON questions (contested_at DESC); /*取最新 100 条争议直接走它*/
