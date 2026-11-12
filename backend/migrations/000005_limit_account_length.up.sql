/*评审确认：账号上限是 7 个字符，不是 64*/
UPDATE users SET account = CASE account
    WHEN 'seed_alice' THEN 'alice01'
    WHEN 'seed_bob' THEN 'bob01'
    WHEN 'seed_carol' THEN 'carol01'
    ELSE account END
  WHERE account IN ('seed_alice', 'seed_bob', 'seed_carol'); /*旧 seed 账号名 8~10 位，不先改名就通不过下面的列宽收紧*/

ALTER TABLE users ALTER COLUMN account TYPE VARCHAR(7);