/*更正 000005：账号上限是 64 位；7 位是昵称的上限，评审时把对象安错了*/
ALTER TABLE users ALTER COLUMN account TYPE VARCHAR(64);
