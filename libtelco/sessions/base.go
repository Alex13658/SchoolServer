// Copyright (C) 2018 Mikhail Masyagin

/*
Package sessions - данный файл содержит в себе сессии на серверах школ.
*/
package sessions

import (
	cp "SchoolServer/libtelco/config-parser"
	dt "SchoolServer/libtelco/sessions/data-types"
	ss "SchoolServer/libtelco/sessions/session"
	t01 "SchoolServer/libtelco/sessions/type-01"
	"fmt"

	gr "github.com/levigross/grequests"
)

type Session struct {
	Base *ss.Session
}

// NewSession создает новую сессию на базе информации о школьном сервере,
// к которому предстоит подключиться.
func NewSession(server *cp.School) *Session {
	return &Session{
		Base: &ss.Session{
			Sess: gr.NewSession(nil),
			Serv: server,
		},
	}
}

/*
Вход в систему.
*/

// Login логинится к серверу и создает очередную сессию.
func (s *Session) Login() error {
	s.Base.MU.Lock()
	defer s.Base.MU.Unlock()
	var err error
	switch s.Base.Serv.Type {
	case cp.FirstType:
		err = t01.Login(s.Base)
	default:
		err = fmt.Errorf("Unknown SchoolServer Type: %d", s.Base.Serv.Type)
	}
	return err
}

/*
Получение списка детей.
*/

// GetChildrenMap получает мапу детей в их UID.
func (s *Session) GetChildrenMap() error {
	s.Base.MU.Lock()
	defer s.Base.MU.Unlock()
	var err error
	switch s.Base.Serv.Type {
	case cp.FirstType:
		err = t01.GetChildrenMap(s.Base)
	default:
		err = fmt.Errorf("Unknown SchoolServer Type: %d", s.Base.Serv.Type)
	}
	return err
}

/*
Получение списка предметов.
*/

// ПЕРЕДЕЛАТЬ!!!

// GetLessonsMap возвращает список пар мапу предметов в их ID.
func (s *Session) GetLessonsMap(studentID string) (*dt.LessonsMap, error) {
	s.Base.MU.Lock()
	defer s.Base.MU.Unlock()
	var err error
	var lessonsMap *dt.LessonsMap
	switch s.Base.Serv.Type {
	case cp.FirstType:
		lessonsMap, err = t01.GetLessonsMap(s.Base, studentID)
	default:
		err = fmt.Errorf("Unknown SchoolServer Type: %d", s.Base.Serv.Type)
	}
	return lessonsMap, err
}