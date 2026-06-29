package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// UseGlobalMiddlewares attaches default global middleware handlers to the given Echo instance.
func UseGlobalMiddlewares(e *echo.Echo) {
	/*
		The order of middleware is important.
		Recover should be the first to catch panics.
		requestIDMiddleware should be early to assign request IDs.
		corsMiddleware should be before any handlers that need CORS.
		loggerMiddleware should log all requests.
	*/
	e.Use(
		middleware.Recover(),
		requestIDMiddleware,
		corsMiddleware(),
		loggerMiddleware(),
	)
}

// UseCustomErrorHandler sets a custom error handler for the given Echo instance.
func UseCustomErrorHandler(e *echo.Echo) {
	e.HTTPErrorHandler = customErrorHandler()
}
