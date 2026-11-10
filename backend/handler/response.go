package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 200, Msg: "success", Data: data})
}

func Error(c *gin.Context, code int, msg string) {
	c.JSON(code, Response{Code: code, Msg: msg})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{Code: http.StatusCreated, Msg: "success", Data: data})
}