package controllers

import (
	"zedlancer/backend/internal/domain/exceptions"
	"zedlancer/backend/internal/infrastructure/validation"	
	"github.com/gin-gonic/gin"
)

func Receive[T any](ctx *gin.Context) T {
	var params T

	if err := ctx.ShouldBind(&params); err != nil {
		bindingErr := exceptions.NewBindingError(err)
		panic(bindingErr)
	}

	if err := ctx.ShouldBindUri(&params); err != nil {
		bindingErr := exceptions.NewBindingError(err)
		panic(bindingErr)
	}

	if err := ctx.ShouldBindQuery(&params); err != nil {
		bindingErr := exceptions.NewBindingError(err)
		panic(bindingErr)
	}

	validation.ValidateFields(params)

	return params
}
