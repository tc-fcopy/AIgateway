package http_proxy_pipeline

import (
	"reflect"
	"sync"
	"unsafe"

	"github.com/gin-gonic/gin"
)

var ginContextLayout sync.Once
var handlersOffset uintptr
var indexOffset uintptr
var handlersOffsetOk bool
var indexOffsetOk bool

func runHandlersChain(c *gin.Context, handlers gin.HandlersChain) {
	if c == nil || len(handlers) == 0 {
		return
	}
	initGinContextLayout()

	oldHandlers := getContextHandlers(c)
	oldIndex := getContextIndex(c)
	defer func() {
		setContextHandlers(c, oldHandlers)
		setContextIndex(c, oldIndex)
	}()

	setContextHandlers(c, handlers)
	setContextIndex(c, -1)
	c.Next()
}

func initGinContextLayout() {
	ginContextLayout.Do(func() {
		t := reflect.TypeOf(gin.Context{})
		handlersField, ok := t.FieldByName("handlers")
		if ok {
			handlersOffset = handlersField.Offset
			handlersOffsetOk = true
		}
		indexField, ok := t.FieldByName("index")
		if ok {
			indexOffset = indexField.Offset
			indexOffsetOk = true
		}
	})
	if !handlersOffsetOk || !indexOffsetOk {
		panic("gin.Context layout changed: handlers/index field not found")
	}
}

func getContextHandlers(c *gin.Context) gin.HandlersChain {
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(c)) + handlersOffset)
	return *(*gin.HandlersChain)(ptr)
}

func setContextHandlers(c *gin.Context, handlers gin.HandlersChain) {
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(c)) + handlersOffset)
	*(*gin.HandlersChain)(ptr) = handlers
}

func getContextIndex(c *gin.Context) int8 {
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(c)) + indexOffset)
	return *(*int8)(ptr)
}

func setContextIndex(c *gin.Context, index int8) {
	ptr := unsafe.Pointer(uintptr(unsafe.Pointer(c)) + indexOffset)
	*(*int8)(ptr) = index
}
