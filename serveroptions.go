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
			errorString := fmt.Sprintf("Requested root directory (%s) does not exist\n", rootDirectory)
			panic(errorString)
		}
		options.RootDirectory = &rootDirectory
	} else {
		rootDirectory, err := os.Getwd()
		if err != nil {
			panic("Unable to get CWD\n")
		}
		options.RootDirectory = &rootDirectory
	}

	headerTemplatePath := os.Getenv("REMOTE_PDF_DEBUG_HEADER_STYLE_TEMPLATE")
	if headerTemplatePath != "" {
		if !pathExists(headerTemplatePath) {
			panic("Unable to locate header path\n")
		}
	} else {
		headerTemplatePath = *options.RootDirectory + "/css/default-header.css.txt"
	}

	headerTemplateBytes, err := os.ReadFile(headerTemplatePath)
	if err != nil {
		errorString := fmt.Sprintf("%s\n", err.Error())
		panic(errorString)
	}

	options.HeaderStyleTemplate = string(headerTemplateBytes) // convert content to a 'string'

	debug := os.Getenv("REMOTE_PDF_DEBUG")
	if debug != "" {
		boolVal, err := strconv.ParseBool(debug)
		if err != nil {
			panic("Unable to parse env REMOTE_PDF_DEBUG\n")
		}

		options.Debug = boolVal
		options.DebugSources = boolVal

		if options.Debug {
			fmt.Printf("Setting debug and debug_sources to %t\n", boolVal)
		}
	}

	debug = os.Getenv("REMOTE_PDF_DEBUG_SOURCES")
	if debug != "" {
		boolVal, err := strconv.ParseBool(debug)
		if err != nil {
			panic("Unable to parse env REMOTE_PDF_DEBUG_SOURCES\n")
		}

		if options.Debug {
			fmt.Printf("Setting debug_sources to %t\n", boolVal)
		}

		options.DebugSources = boolVal
	}

	port := os.Getenv("REMOTE_PDF_PORT")
	if port != "" {
		intPort, err := strconv.ParseInt(port, 10, 16)
		if err != nil {
			panic("Unable to parse env REMOTE_PDF_PORT\n")
		}

		if options.Debug {
			fmt.Printf("Setting port to %d", intPort)
		}
		options.Port = int16(intPort)
	}

	address := os.Getenv("REMOTE_PDF_LISTEN")
	if address != "" {
		if options.Debug {
			fmt.Printf("Setting address to %s\n", address)
		}

		options.Address = address
	}

	address = os.Getenv("REMOTE_PDF_CHROME_URI")
	if address != "" {
		if options.Debug {
			fmt.Printf("Setting chrome uri to %s\n", address)
		}

		options.ChromeUri = address
	}

	useTls := os.Getenv("REMOTE_PDF_TLS_ENABLE")
	if useTls != "" {
		boolVal, err := strconv.ParseBool(useTls)
		if err != nil {
			panic("Unable to parse env REMOTE_PDF_USE_TLS\n")
		}

		if options.Debug {
			fmt.Printf("Setting UseTLS to %t\n", boolVal)
		}

		options.UseTLS = boolVal
	}

	certDir := os.Getenv("REMOTE_PDF_TLS_CERT_DIR")
	if certDir == "" {
		certDir = *options.RootDirectory + "/certs"
	}

	if options.Debug {
		fmt.Printf("Setting CertDirectory to %s\n", certDir)
	}

	options.CertDirectory = &certDir

	if options.UseTLS {
		certPath := os.Getenv("REMOTE_PDF_TLS_CERT_PATH")
		if certPath == "" || (certPath != "" && !pathExists(certPath)) {
			panic("TLS enabled but unable to locate certificate path\n")
		}
		options.CertPath = &certPath

		keyPath := os.Getenv("REMOTE_PDF_TLS_KEY_PATH")
		if keyPath == "" || (keyPath != "" && !pathExists(keyPath)) {
			panic("TLS enabled but unable to locate key path\n")
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
