package web

import (
	"crypto/tls"
	"flag"
	"github.com/NYTimes/gziphandler"
	log "github.com/Sirupsen/logrus"
	"github.com/golang/crypto/acme/autocert"
	"github.com/jonas747/yagpdb/bot/botrest"
	"github.com/jonas747/yagpdb/common"
	"github.com/natefinch/lumberjack"
	"goji.io"
	"goji.io/pat"
	"html/template"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var (
	// Core template files
	Templates *template.Template

	Debug              = true // Turns on debug mode
	ListenAddressHTTP  = ":5000"
	ListenAddressHTTPS = ":5001"

	LogRequestTimestamps bool

	RootMux         *goji.Mux
	CPMux           *goji.Mux
	ServerPublicMux *goji.Mux

	properAddresses bool

	acceptingRequests *int32
)

func init() {
	b := int32(1)
	acceptingRequests = &b

	Templates = template.New("")
	Templates = Templates.Funcs(template.FuncMap{
		"dict":       dictionary,
		"mTemplate":  mTemplate,
		"in":         in,
		"adjective":  common.RandomAdjective,
		"title":      strings.Title,
		"hasPerm":    hasPerm,
		"formatTime": prettyTime,
		"plus1":      plus1,
		"roleAbove":  roleIsAbove,
	})

	Templates = template.Must(Templates.ParseFiles("templates/index.html", "templates/cp_main.html", "templates/cp_nav.html", "templates/cp_selectserver.html", "templates/cp_logs.html"))

	flag.BoolVar(&properAddresses, "pa", false, "Sets the listen addresses to 80 and 443")
}

func Run() {

	if properAddresses {
		ListenAddressHTTP = ":80"
		ListenAddressHTTPS = ":443"
	}

	log.Info("Starting yagpdb web server http:", ListenAddressHTTP, ", and https:", ListenAddressHTTPS)

	InitOauth()
	mux := setupRoutes()

	// Start monitoring the bot
	go botrest.RunPinger()

	log.Info("Running webservers")
	runServers(mux)
}

func Stop() {
	atomic.StoreInt32(acceptingRequests, 0)
}

func IsAcceptingRequests() bool {
	return atomic.LoadInt32(acceptingRequests) != 0
}

func runServers(mainMuxer *goji.Mux) {
	// launch the redir server
	go func() {
		unsafeHandler := &http.Server{
			Addr:        ListenAddressHTTP,
			Handler:     http.HandlerFunc(httpsRedirHandler),
			IdleTimeout: time.Minute,
		}
		err := unsafeHandler.ListenAndServe()
		if err != nil {
			log.Error("Failed http ListenAndServe:", err)
		}
	}()

	cache := autocert.DirCache("cert")

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(common.Conf.Host),
		Email:      common.Conf.Email,
		Cache:      cache,
	}

	tlsServer := &http.Server{
		Addr:        ListenAddressHTTPS,
		Handler:     mainMuxer,
		IdleTimeout: time.Minute,
		TLSConfig: &tls.Config{
			GetCertificate: certManager.GetCertificate,
		},
	}

	err := tlsServer.ListenAndServeTLS("", "")
	if err != nil {
		log.Error("Failed https ListenAndServeTLS:", err)
	}
}

func setupRoutes() *goji.Mux {
	requestLogger := &lumberjack.Logger{
		Filename: "access.log",
		MaxSize:  10,
	}

	mux := goji.NewMux()
	RootMux = mux
	mux.Use(RequestLogger(requestLogger))

	// Setup fileserver
	mux.Handle(pat.Get("/static/*"), http.FileServer(http.Dir(".")))

	// General middleware
	mux.Use(gziphandler.GzipHandler)
	mux.Use(MiscMiddleware)
	mux.Use(RedisMiddleware)
	mux.Use(BaseTemplateDataMiddleware)
	mux.Use(SessionMiddleware)
	mux.Use(UserInfoMiddleware)

	// General handlers
	mux.Handle(pat.Get("/"), RenderHandler(nil, "index"))
	mux.HandleFunc(pat.Get("/login"), HandleLogin)
	mux.HandleFunc(pat.Get("/confirm_login"), HandleConfirmLogin)
	mux.HandleFunc(pat.Get("/logout"), HandleLogout)

	// The public muxer, for public server stuff like stats and logs
	serverPublicMux := goji.SubMux()
	serverPublicMux.Use(ActiveServerMW)

	mux.Handle(pat.Get("/public/:server"), serverPublicMux)
	mux.Handle(pat.Get("/public/:server/*"), serverPublicMux)
	ServerPublicMux = serverPublicMux

	// Server selection has it's own handler
	mux.Handle(pat.Get("/cp"), RenderHandler(HandleSelectServer, "cp_selectserver"))
	mux.Handle(pat.Get("/cp/"), RenderHandler(HandleSelectServer, "cp_selectserver"))

	// Server control panel, requires you to be an admin for the server (owner or have server management role)
	serverCpMuxer := goji.SubMux()
	serverCpMuxer.Use(RequireSessionMiddleware)
	serverCpMuxer.Use(ActiveServerMW)
	serverCpMuxer.Use(RequireServerAdminMiddleware)

	mux.Handle(pat.New("/cp/:server"), serverCpMuxer)
	mux.Handle(pat.New("/cp/:server/*"), serverCpMuxer)

	serverCpMuxer.Handle(pat.Get("/cplogs"), RenderHandler(HandleCPLogs, "cp_action_logs"))
	serverCpMuxer.Handle(pat.Get("/cplogs/"), RenderHandler(HandleCPLogs, "cp_action_logs"))
	CPMux = serverCpMuxer

	for _, plugin := range Plugins {
		plugin.InitWeb()
		log.Info("Initialized web plugin:", plugin.Name())
	}

	return mux
}

func httpsRedirHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
}
