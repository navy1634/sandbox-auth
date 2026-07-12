package handler

import "github.com/gin-gonic/gin"

type AuthMethodHandler interface {
	RegisterRoutes(routes gin.IRoutes)
}
