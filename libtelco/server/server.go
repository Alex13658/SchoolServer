// Copyright (C) 2018 Mikhail Masyagin

/*
Package server содержит основную функциональность сервера.
*/
package server

import (
	cp "SchoolServer/libtelco/config-parser"
	"SchoolServer/libtelco/log"

	api "SchoolServer/libtelco/rest-api"
	//ss "SchoolServer/libtelco/sessions"

	"net/http"

	"runtime"
)

// Server struct содержит конфигурацию сервера.
type Server struct {
	config *cp.Config
	api    *api.RestAPI
	logger *log.Logger
}

// NewServer создает новый сервер.
func NewServer(config *cp.Config, logger *log.Logger) *Server {
	serv := &Server{
		config: config,
		api:    api.NewRestAPI(logger, config),
	}
	return serv
}

// Run запускает сервер.
func (serv *Server) Run() error {
	// Задаем максимальное количество потоков.
	runtime.GOMAXPROCS(serv.config.MaxProcs)

	/*
		// Тесты.
		kek := ss.NewSession(&serv.config.Schools[2])

		err := kek.Login()
		if err != nil {
			fmt.Println(err)
		}

		data, err := kek.GetResourcesList()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(data)
	*/

	// Подключаем handler'ы из RestAPI.
	serv.api.BindHandlers()
	return http.ListenAndServe(serv.config.ServerAddr, nil)
}
