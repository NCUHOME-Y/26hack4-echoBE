package model

//经验值来源类型，写入 exp_entries.source_type
const (
	ExpSourceQuestion = "question" //发布问题
	ExpSourceVote     = "vote"     //投票作答
	ExpSourceSettle   = "settle"   //问题结算
)

//经验值变动原因，写入 exp_entries.reason
const (
	ExpReasonAsk          = "ask"           //发布问题
	ExpReasonAnswer       = "answer"        //回答随机问题
	ExpReasonSettleAsk    = "settle_ask"    //给提问者的结算奖励
	ExpReasonSettleMatch  = "settle_match"  //与提问者选择一致
	ExpReasonSettleMiss   = "settle_miss"   //与提问者选择不一致
	ExpReasonSettleFailed = "settle_failed" //结算失败（未选择最终情况）补偿
)

//积分规则数值
const (
	ExpAsk          = 10 //发布问题
	ExpAnswer       = 5  //回答随机问题
	ExpSettleAsk    = 40 //提问者结算奖励，无人回复也发
	ExpSettleMatch  = 25 //与提问者选择一致
	ExpSettleMiss   = 5  //与提问者选择不一致
	ExpSettleFailed = 15 //结算失败补偿投票者
	LevelExpUnit    = 50 //50 积分一级
	MaxLevel        = 9999
)
