/*清空数据库*/
TRUNCATE TABLE questions, options, votes, question_skips, replies, exp_entries
    RESTART IDENTITY CASCADE;

/*1) 测试用户：hash 是真实 bcrypt 值，统一密码 Echo12345678（符合 min=8,max=64,alphanum）
   这四个账号可以直接登录，演示和自测不用先注册一次性账号*/
INSERT INTO users (id, account, nickname, password_hash, exp) OVERRIDING SYSTEM VALUE VALUES
  (9001, 'alice01', 'Alice', '$2a$10$8h5Yv933snTCl2gvWZwGXOCjQXOUIQuAHKnjBTdS78vl8.YkRijim', 0),
  (9002, 'bob01',   'Bob',   '$2a$10$8h5Yv933snTCl2gvWZwGXOCjQXOUIQuAHKnjBTdS78vl8.YkRijim', 0),
  (9003, 'carol01', 'Carol', '$2a$10$8h5Yv933snTCl2gvWZwGXOCjQXOUIQuAHKnjBTdS78vl8.YkRijim', 0),
  (9004, 'dave01',  'Dave',  '$2a$10$8h5Yv933snTCl2gvWZwGXOCjQXOUIQuAHKnjBTdS78vl8.YkRijim', 0)
ON CONFLICT (id) DO UPDATE SET password_hash = EXCLUDED.password_hash;

/*2) 状态覆盖：未到期 / 即将到期 / 已过期未结算 / 已结算非争议 / 已结算且入争议库
   status 只有 open 和 closed 两种，争议一律看 contested_at，不再用历史遗留的 contested 值*/
INSERT INTO questions (id, asker_id, description, status, option_count, expires_at, settled_at, contested_at) OVERRIDING SYSTEM VALUE VALUES
  (9001, 9001, '周末去图书馆还是在家赶作业',   'open',   2, now() + interval '7 days', NULL,                          NULL),
  (9002, 9002, '社团面试和室友生日撞了选哪个', 'open',   3, now() + interval '2 hours', NULL,                          NULL),
  (9003, 9003, '换新手机还是攒钱换电脑',       'open',   2, now() - interval '1 hour',  NULL,                          NULL),
  (9004, 9001, '考研还是直接工作',             'closed', 2, now() - interval '3 days',  now() - interval '3 days' - interval '1 hour', NULL),
  (9005, 9002, '午饭吃食堂还是点外卖',         'closed', 2, now() - interval '1 day',   now() - interval '1 day' - interval '1 hour', now() - interval '1 day' - interval '1 hour');

/*3) 选项：题 9002 给三个选项，题 9004 和 9005 故意做成票数接近*/
INSERT INTO options (id, question_id, label, sort_no) OVERRIDING SYSTEM VALUE VALUES
  (9001, 9001, '图书馆', 1), (9002, 9001, '在家', 2),
  (9003, 9002, '面试',   1), (9004, 9002, '生日', 2), (9005, 9002, '都推掉', 3),
  (9006, 9003, '买手机', 1), (9007, 9003, '换电脑', 2),
  (9008, 9004, '考研',   1), (9009, 9004, '工作', 2),
  (9010, 9005, '食堂',   1), (9011, 9005, '外卖', 2);

/*4) 已结算的两道题带上最终选择（必须是它自己的选项，否则复合外键会拦）
   最终选择只由提问者本人在到期前用 #13 指定，后端不按票数猜；到期时没指定就永久为空，事后不开放补公布。
   所以这里非空的 final_option_id 代表的是"到期前就公布了"，上面两题的 settled_at 必须早于 expires_at，
   否则就是代码产不出来的状态。要演示"到期了但没定"，把这两行注释掉即可*/
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

/*8) 经验流水：代表"第 9 段那两条题结算完成之后"的库状态，reason 和 delta 全部取 model/exp.go 的常量
   ask=10 每题一次 / settle_ask=40 只给到期前定了最终选择的提问者 / settle_match=25 与最终选择一致 / settle_miss=5 不一致
   改常量要同步改这里，否则余额和流水对不上，第 9 段重算出来的 exp 就露馅了
   唯一约束是 (user_id, source_type, source_id, reason)，同一题同一理由不能出现两次
   9001 9002 9003 三道题还没结算，所以只有 ask，没有 settle 系列*/
INSERT INTO exp_entries (user_id, delta, reason, source_type, source_id) VALUES
  (9001, 10, 'ask', 'question', 9001), (9001, 10, 'ask', 'question', 9004),
  (9002, 10, 'ask', 'question', 9002), (9002, 10, 'ask', 'question', 9005),
  (9003, 10, 'ask', 'question', 9003),
  /*9004 final=9008：提问者 9001 +40，投 9008 的 9002 匹配 +25，投 9009 的 9003 不匹配 +5*/
  (9001, 40, 'settle_ask',   'question', 9004),
  (9002, 25, 'settle_match', 'question', 9004),
  (9003,  5, 'settle_miss',  'question', 9004),
  /*9005 final=9010：提问者 9002 +40，投 9010 的 9001 匹配 +25，投 9011 的 9003 不匹配 +5*/
  (9002, 40, 'settle_ask',   'question', 9005),
  (9001, 25, 'settle_match', 'question', 9005),
  (9003,  5, 'settle_miss',  'question', 9005);

/*9) 余额必须等于流水之和*/
UPDATE users u SET exp = COALESCE((SELECT SUM(e.delta) FROM exp_entries e WHERE e.user_id = u.id), 0)
WHERE u.id IN (9001, 9002, 9003, 9004);

/*10) 关于 9005 的争议态：contested_at 是直接写进去的，代表"判定逻辑跑完之后"的库状态
   它只有 2 票，不满足总票数 >= 30 的门槛，所以只能用来验争议列表的查询，不能用来验判定函数本身
   要验判定：临时把阈值调小（记得改回来），或者另造 30 个用户灌票*/
