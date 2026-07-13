package handler

import "github.com/gin-gonic/gin"

type AuthMethodHandler interface {
	// 認証方式ごとに必要なルートを Gin に登録する。
	RegisterRoutes(routes gin.IRoutes)
}
