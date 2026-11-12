/*问题投票系统核心表，用来保存用户发表的问题*/
CREATE TABLE questions (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, /*自增主键*/
    asker_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE, /*用户被删除时，问题也会被删除*/
    description VARCHAR(102)        NOT NULL CHECK (char_length(description) BETWEEN 1 AND 102), /*按字符数计算，长度必须在1到102个字符之间*/
    status      VARCHAR(16)  NOT NULL DEFAULT 'open'
                             CHECK (status IN ('open','contested','closed')),/*只允许这三种状态*/
    expires_at  TIMESTAMPTZ  NOT NULL, /*问题截止时间，且为带时区的时间*/
    settled_at  TIMESTAMPTZ, /*非空表示已结算，结算任务重复跑时靠它跳过，保证幂等*/
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(), /*问题创建时间，默认为现在的时间*/
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now() /*问题更新时间，默认为现在的时间*/
);
CREATE INDEX questions_asker_idx      ON questions (asker_id, id DESC); /*根据提问者和问题ID降序排列，可以查找某个用户提出的最新问题*/
CREATE INDEX questions_status_exp_idx ON questions (status, expires_at); /*根据问题状态和截止时间创建索引，用于快速查找即将过期或某种状态的问题*/

/*问题选项表*/
CREATE TABLE options (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    question_id BIGINT       NOT NULL REFERENCES questions(id) ON DELETE CASCADE, /*所属于哪个问题*/
    label       VARCHAR(18) NOT NULL, /*选项文字*/
    sort_no     SMALLINT     NOT NULL, /*选项显示顺序*/
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(), 
    UNIQUE (question_id, sort_no), /*同一个问题下，选项序号不能重复*/
    UNIQUE (id, question_id) /*保证每一组选项和问题唯一对应*/
);

/*给问题增加最终选项*/
ALTER TABLE questions
    ADD COLUMN final_option_id BIGINT REFERENCES options(id) ON DELETE SET NULL; /*最终选项被删除不会删除问题*/

/*用户投票记录表表*/
CREATE TABLE votes (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    question_id BIGINT      NOT NULL REFERENCES questions(id) ON DELETE CASCADE, /*投票所属的问题，且会随问题删除而删除*/
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    option_id   BIGINT      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (question_id, user_id),
    CONSTRAINT votes_option_belongs /*外键约束命名*/
        FOREIGN KEY (option_id, question_id) REFERENCES options (id, question_id) /*确保投票的选项属于该问题*/
);
CREATE INDEX votes_user_idx ON votes (user_id, id DESC); /*查询用户投票记录*/

/*被跳过的问题表，保证一个问题被跳过后不会再出现*/
CREATE TABLE question_skips (
    question_id BIGINT      NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skipped_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (question_id, user_id) /*将用户ID和问题ID作为联合主键*/
);

/*问题回复表（评论区）*/
CREATE TABLE replies (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    question_id BIGINT      NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT        NOT NULL CHECK (char_length(content) BETWEEN 1 AND 32),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(), /*回复最后更新时间，为了应对特殊情况，前端不用开放更新评论的功能*/
    deleted_at  TIMESTAMPTZ /*软删除，只是在用户那边删除，数据库中保留数据*/
);
CREATE INDEX replies_question_idx ON replies (question_id, created_at); /*以问题ID和创建时间作为索引，方便按时间查找某个问题的回复*/

ALTER TABLE users ADD COLUMN exp INT NOT NULL DEFAULT 0 CHECK (exp >= 0);

/*经验流水表*/
CREATE TABLE exp_entries (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delta       INT         NOT NULL CHECK (delta <> 0), /*经验值变化，不能为0*/
    reason      VARCHAR(32) NOT NULL,
    source_type VARCHAR(16) NOT NULL,
    source_id   BIGINT      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, source_type, source_id, reason) /*保证同一个用户对同一个来源的经验流水不会重复*/
);
CREATE INDEX exp_entries_user_idx ON exp_entries (user_id, id DESC); /*以用户ID和ID降序作为索引，方便按用户查询其经验流水*/