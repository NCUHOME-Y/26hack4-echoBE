/*清空数据库*/
TRUNCATE TABLE questions, options, votes, question_skips, replies, exp_entries
    RESTART IDENTITY CASCADE;

/*1) 测试用户：占位 hash 不能登录，只用于外键和聚合*/
INSERT INTO users (id, account, nickname, password_hash, exp) OVERRIDING SYSTEM VALUE VALUES
  (9001, 'seed_alice', 'Alice', 'x', 0),
  (9002, 'seed_bob',   'Bob',   'x', 0),
  (9003, 'seed_carol', 'Carol', 'x', 0)
ON CONFLICT (id) DO NOTHING;

/*2) 五种状态各一条，覆盖后面要测的每条分支*/
INSERT INTO questions (id, asker_id, description, status, option_count, expires_at, settled_at) OVERRIDING SYSTEM VALUE VALUES
  (9001, 9001, '周末去图书馆还是在家赶作业',   'open',      2, now() + interval '7 days',  NULL),
  (9002, 9002, '社团面试和室友生日撞了选哪个', 'open',      3, now() + interval '2 hours', NULL),
  (9003, 9003, '换新手机还是攒钱换电脑',       'open',      2, now() - interval '1 hour',  NULL),
  (9004, 9001, '考研还是直接工作',             'closed',    2, now() - interval '3 days',  now() - interval '10 minutes'),
  (9005, 9002, '午饭吃食堂还是点外卖',         'contested', 2, now() + interval '5 days',  NULL);

/*3) 选项：题 9002 给三个选项，题 9005 故意做成票数接近*/
INSERT INTO options (id, question_id, label, sort_no) OVERRIDING SYSTEM VALUE VALUES
  (9001, 9001, '图书馆', 1), (9002, 9001, '在家', 2),
  (9003, 9002, '面试',   1), (9004, 9002, '生日', 2), (9005, 9002, '都推掉', 3),
  (9006, 9003, '买手机', 1), (9007, 9003, '换电脑', 2),
  (9008, 9004, '考研',   1), (9009, 9004, '工作', 2),
  (9010, 9005, '食堂',   1), (9011, 9005, '外卖', 2);

/*4) 已结算的题补上最终选择（必须是它自己的选项，否则复合外键会拦）*/
UPDATE questions SET final_option_id = 9008 WHERE id = 9004;

/*5) 投票：一人一题一票，分布不均*/
INSERT INTO votes (question_id, user_id, option_id) VALUES
  (9001, 9002, 9001), (9001, 9003, 9002),
  (9002, 9001, 9003), (9002, 9003, 9004),
  (9003, 9001, 9006), (9003, 9002, 9007), (9003, 9003, 9006),
  (9004, 9002, 9008), (9004, 9003, 9009),
  (9005, 9001, 9010), (9005, 9003, 9011);

/*6) 跳过记录：9002 跳过 9001，抽题时它不该再出现*/
INSERT INTO question_skips (question_id, user_id) VALUES (9001, 9002);

/*7) 评论：短小即可，上限探测属于约束测试，不混进 seed*/
INSERT INTO replies (question_id, user_id, content) VALUES
  (9001, 9002, '在家效率其实更高，前提是手机放远点'),
  (9002, 9003, '两个都不能推，看看能不能错峰');

/*8) 经验流水：reason 是幂等键的一部分，占位先统一用 settle*/
INSERT INTO exp_entries (user_id, delta, reason, source_type, source_id) VALUES
  (9002, 20, 'settle', 'question', 9004),
  (9003, 20, 'settle', 'question', 9004),
  (9002, 35, 'settle', 'question', 9001);

/*9) 余额必须等于流水之和*/
UPDATE users u SET exp = COALESCE((SELECT SUM(e.delta) FROM exp_entries e WHERE e.user_id = u.id), 0)
WHERE u.id IN (9001, 9002, 9003);
