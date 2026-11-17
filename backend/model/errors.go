package model

import "errors"

// 哨兵错误：上层用 errors.Is 判断，不用比字符串
var ErrAccountTaken = errors.New("账号已被注册")
var ErrUserNotFound = errors.New("用户不存在")
var ErrInvalidCredentials = errors.New("账号或密码错误")
var ErrInvalidAvatar = errors.New("无效的头像标识")
var ErrQuestionNotFound = errors.New("问题不存在")          //404，抽题详情投票通用
var ErrOptionNotInQuestion = errors.New("选项不属于该问题")    //404，投票和公布结果共用
var ErrAlreadyVoted = errors.New("你已经投过这一题了")          //409，重复投票
var ErrQuestionSkipped = errors.New("你已放弃这一题的作答权")     //409，跳过之后再来投票
var ErrQuestionClosed = errors.New("这道题已经结束，不能再投票")    //409，已关闭或已过期
var ErrOwnQuestionVote = errors.New("自己提的题不能投票")       //409
var ErrOwnQuestionSkip = errors.New("自己提的题不用跳过")       //409
var ErrAlreadyVotedSkip = errors.New("你已经投过这一题，不能再跳过") //409，跳过接口专用，与 ErrAlreadyVoted 原文不同
var ErrInvalidDescription = errors.New("问题描述不合法")
var ErrInvalidOptionCount = errors.New("选项数量不合法")
var ErrInvalidOptionLabel = errors.New("选项文字不合法")
var ErrInvalidDuration = errors.New("结算档位不合法")
var ErrInvalidReplyContent = errors.New("评论内容不合法")           //400，发表回复，binding 之后还要按 trim 结果再判一次
var ErrReplyNotFound = errors.New("回复不存在")                   //404，删除回复：不存在与已删过走同一句
var ErrNotReplyOwner = errors.New("只能删除自己的回复，或由提问者删除本题下的回复") //403，删除回复
var ErrNotAsker = errors.New("只有提问者能公布最终选择")                 //403，公布最终选择
var ErrFinalAlreadySet = errors.New("这道题的最终选择已经定过了，不能再修改")   //409，重复公布
var ErrFinalWindowClosed = errors.New("这道题已经结束，不能再公布最终选择") //409，到期后不再开放补公布
