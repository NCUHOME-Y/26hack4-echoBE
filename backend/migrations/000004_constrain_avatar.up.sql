/*给 avatar 加数据库层约束：长度上限交给列类型，格式用 CHECK 拦*/
ALTER TABLE users ALTER COLUMN avatar TYPE VARCHAR(32); /*上限的权威来源是列类型，和 binding 的 max=32 对齐*/
ALTER TABLE users ADD CONSTRAINT users_avatar_check CHECK (avatar = '' OR avatar ~ '^avatar-[0-9]{1,3}$'); /*空串留给"未设置头像"的存量数据和注册默认值；等内置头像标识定下来后应把默认值改成 avatar-N 并去掉空串分支*/