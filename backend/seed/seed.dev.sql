/*清空数据库*/
TRUNCATE TABLE questions, options, votes, question_skips, replies, exp_entries
    RESTART IDENTITY CASCADE;

/*1) 测试用户：占位 hash 不能登录，只用于外键和聚合*/
INSERT INTO users (id, account, nickname, password_hash, exp) OVERRIDING SYSTEM VALUE VALUES
  (9001, 'alice01', 'Alice', 'x', 0),
  (9002, 'bob01',   'Bob',   'x', 0),
  (9003, 'carol01', 'Carol', 'x', 0),
  (9004, 'dave01',  'Dave',  'x', 0)
ON CONFLICT (id) DO NOTHING;

/*2) 状态覆盖：未到期 / 即将到期 / 已过期未结算 / 已结算非争议 / 已结算且入争议库
   status 只有 open 和 closed 两种，争议一律看 contested_at，不再用历史遗留的 contested 值*/
INSERT INTO questions (id, asker_id, description, status, option_count, expires_at, settled_at, contested_at) OVERRIDING SYSTEM VALUE VALUES
  (9001, 9001, '周末去图书馆还是在家赶作业',   'open',   2, now() + interval '7 days', NULL,                          NULL),
  (9002, 9002, '社团面试和室友生日撞了选哪个', 'open',   3, now() + interval '2 hours', NULL,                          NULL),
  (9003, 9003, '换新手机还是攒钱换电脑',       'open',   2, now() - interval '1 hour',  NULL,                          NULL),
  (9004, 9001, '考研还是直接工作',             'closed', 2, now() - interval '3 days',  now() - interval '10 minutes', NULL),
  (9005, 9002, '午饭吃食堂还是点外卖',         'closed', 2, now() - interval '1 day',   now() - interval '20 minutes', now() - interval '20 minutes');

/*3) 选项：题 9002 给三个选项，题 9004 和 9005 故意做成票数接近*/
INSERT INTO options (id, question_id, label, sort_no) OVERRIDING SYSTEM VALUE VALUES
  (9001, 9001, '图书馆', 1), (9002, 9001, '在家', 2),
  (9003, 9002, '面试',   1), (9004, 9002, '生日', 2), (9005, 9002, '都推掉', 3),
  (9006, 9003, '买手机', 1), (9007, 9003, '换电脑', 2),
  (9008, 9004, '考研',   1), (9009, 9004, '工作', 2),
  (9010, 9005, '食堂',   1), (9011, 9005, '外卖', 2);

/*4) 已结算的两道题补上最终选择（必须是它自己的选项，否则复合外键会拦）
   9004 和 9005 都是 1:1 平票，这里按"sort_no 小者优先"取第一个选项；平票的正式规则还没定*/
UPDATE questions SET final_option_id = 9008 WHERE id = 9004;
UPDATE questions SET final_option_id = 9010 WHERE id = 9005;

/*5) 投票：一人一题一票，分布不均*/
INSERT INTO votes (question_id, user_id, option_id) VALUES
  (9001, 9002, 9001), (9001, 9003, 9002),
  (9002, 9001, 9003), (9002, 9003, 9004),
  (9003, 9001, 9006), (9003, 9002, 9007), (9003, 9003, 9006),
  (9004, 9002, 9008), (9004, 9003, 9009),
  (9005, 9001, 9010), (9005, 9003, 9011);

/*6) 跳过记录：9004 跳过 9001，它没投过任何题，所以这行是合法状态
   原来写成 9002 跳 9001，但 9002 已经投过 9001，而契约规定投过票就不能再跳过
   那是一条逻辑上不可能的记录，测抽题排除时分不清是 votes 还是 question_skips 在起作用*/
INSERT INTO question_skips (question_id, user_id) VALUES (9001, 9004);

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
WHERE u.id IN (9001, 9002, 9003, 9004);

/*10) 关于 9005 的争议态：contested_at 是直接写进去的，代表"判定逻辑跑完之后"的库状态
   它只有 2 票，不满足总票数 >= 30 的门槛，所以只能用来验争议列表的查询，不能用来验判定函数本身
   要验判定：临时把阈值调小（记得改回来），或者另造 30 个用户灌票*/
