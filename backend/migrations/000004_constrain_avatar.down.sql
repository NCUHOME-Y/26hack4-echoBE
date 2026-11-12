/*回退 000004，顺序与 up 相反*/
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_avatar_check;
ALTER TABLE users ALTER COLUMN avatar TYPE TEXT;