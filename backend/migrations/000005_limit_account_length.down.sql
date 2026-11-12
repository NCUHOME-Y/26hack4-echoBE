/*回退 000005：先放宽列宽，再把 seed 账号名换回去*/
ALTER TABLE users ALTER COLUMN account TYPE VARCHAR(64);

UPDATE users SET account = CASE account
    WHEN 'alice01' THEN 'seed_alice'
    WHEN 'bob01' THEN 'seed_bob'
    WHEN 'carol01' THEN 'seed_carol'
    ELSE account END
  WHERE account IN ('alice01', 'bob01', 'carol01');