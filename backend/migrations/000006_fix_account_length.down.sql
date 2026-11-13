/*回退到 7 位：库里若存在超过 7 位的账号，这条 ALTER 会失败，需要先手工清理超长账号*/
ALTER TABLE users ALTER COLUMN account TYPE VARCHAR(7);
