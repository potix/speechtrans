package main

import (
        "encoding/json"
        "flag"
        "github.com/potix/utils/signal"
        "github.com/potix/utils/server"
        "github.com/potix/utils/configurator"
        "github.com/potix/speechtrans/handler"
        "log"
        "log/syslog"
)

type speechtransHttpServerConfig struct {
        Mode        string `toml:"mode"`
        AddrPort    string `toml:"addrPort"`
        TlsCertPath string `toml:"tlsCertPath"`
        TlsKeyPath  string `toml:"tlsKeyPath"`
	SkipVerify  bool   `toml:"skipVerify`
}

type speechtransHttpHandlerConfig struct {
        ResourcePath string            `toml:"resourcePath"`
        Accounts     map[string]string `toml:"accounts"`
        ProjectId    string            `toml:"projectId"`
}

type speechtransLogConfig struct {
        UseSyslog bool `toml:"useSyslog"`
}

type speechtransConfig struct {
        Verbose     bool                          `toml:"verbose"`
        HttpServer  *speechtransHttpServerConfig  `toml:"httpServer"`
        HttpHandler *speechtransHttpHandlerConfig `toml:"httpHandler"`
        Log         *speechtransLogConfig         `toml:"log"`
}

type commandArguments struct {
        configFile string
}

func verboseLoadedConfig(config *speechtransConfig) {
        if !config.Verbose {
                return
        }
        j, err := json.Marshal(config)
        if err != nil {
                log.Printf("can not dump config: %v", err)
                return
        }
        log.Printf("loaded config: %v", string(j))
}

func main() {
        cmdArgs := new(commandArguments)
        flag.StringVar(&cmdArgs.configFile, "config", "./speechtrans.conf", "config file")
        flag.Parse()
        cf, err := configurator.NewConfigurator(cmdArgs.configFile)
        if err != nil {
                log.Fatalf("can not create configurator: %v", err)
        }
        var conf speechtransConfig
        err = cf.Load(&conf)
        if err != nil {
                log.Fatalf("can not load config: %v", err)
        }
        if conf.HttpServer == nil || conf.HttpHandler == nil {
                log.Fatalf("invalid config")
        }
        if conf.Log != nil && conf.Log.UseSyslog {
                logger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "aars")
                if err != nil {
                        log.Fatalf("can not create syslog: %v", err)
                }
                log.SetOutput(logger)
        }
        verboseLoadedConfig(&conf)
	// setup http handler
	hhVerboseOpt := handler.HttpVerbose(conf.Verbose)
	newHttpHandler, err := handler.NewHttpHandler(
                conf.HttpHandler.ResourcePath,
                conf.HttpHandler.Accounts,
		conf.HttpHandler.ProjectId,
                hhVerboseOpt,
        )
        if err != nil {
                log.Fatalf("can not create http handler: %v", err)
        }
	// setup http server
        hsVerboseOpt := server.HttpServerVerbose(conf.Verbose)
        hsTlsOpt := server.HttpServerTls(conf.HttpServer.TlsCertPath, conf.HttpServer.TlsKeyPath)
        hsSkipVerifyOpt := server.HttpServerSkipVerify(conf.HttpServer.SkipVerify)
        hsModeOpt := server.HttpServerMode(conf.HttpServer.Mode)
        newHttpServer, err := server.NewHttpServer(
                conf.HttpServer.AddrPort,
                newHttpHandler,
                hsTlsOpt,
		hsSkipVerifyOpt,
                hsModeOpt,
                hsVerboseOpt,
        )
        if err != nil {
                log.Fatalf("can not create http server: %v", err)
        }
        err = newHttpServer.Start()
        if err != nil {
                log.Fatalf("can not start http server: %v", err)
        }
        signal.SignalWait(nil)
        newHttpServer.Stop()
}
