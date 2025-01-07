package main

import (
	"fmt"
	"net/http/httputil"

	"github.com/gin-gonic/gin"
)

func LogRequestDataMiddleware(serverOptions *ServerOptions) gin.HandlerFunc {

	return func(ctx *gin.Context) {
		fmt.Println(ctx.Request.Host, ctx.Request.RemoteAddr, ctx.Request.RequestURI)

		// Save a copy of this request for debugging.
		requestDump, err := httputil.DumpRequest(ctx.Request, true)
		if err != nil {
			serverOptions.LogFile.WriteString(fmt.Sprintln(err))

			// os.WriteFile(serverOptions.LogFile, , os.ModeAppend)
		}

		serverOptions.LogFile.Write(requestDump)
		serverOptions.LogFile.WriteString("\n")

		ctx.Next()
	}
}
