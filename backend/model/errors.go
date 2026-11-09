package model

import "errors"

// 哨兵错误：上层用 errors.Is 判断，不用比字符串
var ErrAccountTaken = errors.New("账号已被注册")
