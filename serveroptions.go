package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ServerOptions struct {
	Port                int16
	Address             string
	UseTLS              bool
	CertDirectory       *string
	CertPath            *string
	KeyPath             *string
	LogPath             string
	RootDirectory       *string
	HeaderStyleTemplate string
	ChromeUri           string
	Debug               bool
	DebugSources        bool
}

func New(src *ServerOptions) *ServerOptions {
	options := ServerOptions{}
	options.Port = 3000
	options.Address = "127.0.0.1"
	options.UseTLS = true
	options.LogPath = "/var/log"
	options.Debug = false
	options.DebugSources = false
	options.ChromeUri = "127.0.0.1:1337"

	rootDirectory := os.Getenv("REMOTE_PDF_ROOT_DIRECTORY")
	if rootDirectory != "" {
		if !pathExists(rootDirectory) {
			errorString := fmt.Sprintf("Requested root directory (%s) does not exist", rootDirectory)
			panic(errorString)
		}
		options.RootDirectory = &rootDirectory
	} else {
		rootDirectory, err := os.Getwd()
		if err != nil {
			panic("Unable to get CWD")
		}
		options.RootDirectory = &rootDirectory
	}

	headerTemplatePath := os.Getenv("REMOTE_PDF_DEBUG_HEADER_STYLE_TEMPLATE")
	if headerTemplatePath != "" {
		if !pathExists(headerTemplatePath) {
			panic("Unable to locate header path")
		}
	} else {
		headerTemplatePath = *options.RootDirectory + "/css/default-header.css.txt"
	}

	headerTemplateBytes, err := os.ReadFile(headerTemplatePath)
	if err != nil {
		panic(err)
	}

	options.HeaderStyleTemplate = string(headerTemplateBytes) // convert content to a 'string'

	debug := os.Getenv("REMOTE_PDF_DEBUG")
	if debug != "" {
		boolVal, err := strconv.ParseBool(debug)
		if err != nil {
			panic("Unable to parse env REMOTE_PDF_DEBUG")
		}
		options.Debug = boolVal
		options.DebugSources = boolVal
	}

	debug = os.Getenv("REMOTE_PDF_DEBUG_SOURCES")
	if debug != "" {
		boolVal, err := strconv.ParseBool(debug)
		if err != nil {
			panic("Unable to parse env REMOTE_PDF_DEBUG_SOURCEs")
		}
		options.DebugSources = boolVal
	}

	port := os.Getenv("REMOTE_PDF_PORT")
	if port != "" {
		intPort, err := strconv.ParseInt(port, 10, 16)
		if err != nil {
			panic("Unable to parse env REMOTE_PDF_PORT")
		}

		if options.Debug {
			fmt.Printf("Setting port to %d", intPort)
		}
		options.Port = int16(intPort)
	}

	address := os.Getenv("REMOTE_PDF_LISTEN")
	if address != "" {
		if options.Debug {
			fmt.Printf("Setting address to %s", address)
		}

		options.Address = address
	}

	address = os.Getenv("REMOTE_PDF_CHROME_URI")
	if address != "" {
		if options.Debug {
			fmt.Printf("Setting chrome uri to %s", address)
		}

		options.ChromeUri = address
	}

	useTls := os.Getenv("REMOTE_PDF_TLS_ENABLE")
	if useTls != "" {
		boolVal, err := strconv.ParseBool(useTls)
		if err != nil {
			panic("Unable to parse env REMOTE_PDF_USE_TLS")
		}

		if options.Debug {
			fmt.Printf("Setting UseTLS to %t", boolVal)
		}

		options.UseTLS = boolVal
	}

	certDir := os.Getenv("REMOTE_PDF_TLS_CERT_DIR")
	if certDir == "" {
		certDir = *options.RootDirectory + "/certs"
	}

	if options.Debug {
		fmt.Printf("Setting CertDirectory to %s", certDir)
	}

	options.CertDirectory = &certDir

	if options.UseTLS {
		certPath := os.Getenv("REMOTE_PDF_TLS_CERT_PATH")
		if certPath == "" || (certPath != "" && !pathExists(certPath)) {
			panic("TLS enabled but unable to locate certificate path")
		}
		options.CertPath = &certPath

		keyPath := os.Getenv("REMOTE_PDF_TLS_KEY_PATH")
		if keyPath == "" || (keyPath != "" && !pathExists(keyPath)) {
			panic("TLS enabled but unable to locate key path")
		}
		options.KeyPath = &keyPath
	}

	logPath := os.Getenv("REMOTE_PDF_LOG_PATH")
	if logPath != "" {
		options.LogPath = logPath
	}

	return &options
}

// ApiMiddleware will add the ServerOptions connection to the context
func ApiMiddleware(serverOptions *ServerOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("serverOptions", serverOptions)
		c.Next()
	}
}
