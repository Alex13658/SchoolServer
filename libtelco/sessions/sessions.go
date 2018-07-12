// Copyright (C) 2018 Mikhail Masyagin

/*
Package sessions - данный файл содержит в себе сессии на серверах школ.
*/
package sessions

import (
	cp "SchoolServer/libtelco/config-parser"
	"fmt"
	"sync"
	"time"

	gr "github.com/levigross/grequests"
)

// Session struct содержит в себе описание сессии к одному из школьных серверов.
type Session struct {
	// Общая структура.
	sess        *gr.Session
	Serv        *cp.School
	mu          sync.Mutex
	lastRequest time.Time
	// Для серверов первого типа.
	at  string
	ver string
}

// NewSession создает новую сессию на базе информации о школьном сервере,
// к которому предстоит подключиться.
func NewSession(server *cp.School) *Session {
	return &Session{
		sess: nil,
		Serv: server,
		mu:   sync.Mutex{},
	}
}

/*
Получение расписания.
*/

// Login логинится к серверу и создает очередную сессию.
func (s *Session) Login() error {
	var err error
	switch s.Serv.Type {
	case cp.FirstType:
		err = s.firstTypeLogin()
	default:
		err = fmt.Errorf("Unknown SchoolServer Type: %d", s.Serv.Type)
	}
	return err
}

// TimeTable struct содержит в себе расписание на N дней (N = 1, 2, ..., 7).
type TimeTable struct {
	Days []DayTimeTable `json:"days"`
}

// DayTimeTable struct содержит в себе расписание на день.
type DayTimeTable struct {
	Date    string   `json:"date"`
	Lessons []Lesson `json:"lesson"`
}

// Lesson struct содержит в себе один урок.
type Lesson struct {
	Begin     string `json:"begin"`
	End       string `json:"end"`
	Name      string `json:"name"`
	ClassRoom string `json:"classroom"`
}

// GetTimeTable возвращает расписание на n дней, начиная с текущего.
func (s *Session) GetTimeTable(date string, n int) (*TimeTable, error) {
	var err error
	var timeTable *TimeTable
	if (n < 1) || (n > 7) {
		err = fmt.Errorf("Invalid days number")
		return timeTable, err
	}
	timeTable = &TimeTable{
		Days: make([]DayTimeTable, 0, n),
	}
	for i := 0; i < n; i++ {
		day, err := s.getDayTimeTable(date)
		if err != nil {
			return timeTable, err
		}
		timeTable.Days = append(timeTable.Days, *day)
		date, err = incDate(date)
		if err != nil {
			return timeTable, err
		}
	}
	return timeTable, err
}

// getDayTimeTable возвращает расписание на один день.
func (s *Session) getDayTimeTable(date string) (*DayTimeTable, error) {
	var err error
	var dayTimeTable *DayTimeTable
	switch s.Serv.Type {
	case cp.FirstType:
		dayTimeTable, err = s.getDayTimeTableFirst(date)
	default:
		err = fmt.Errorf("Unknown SchoolServer Type: %d", s.Serv.Type)
	}
	return dayTimeTable, err
}

/*
Получение оценок.
*/

// WeekSchoolMarks struct содержит в себе оценки и ДЗ на текущую неделю.
type WeekSchoolMarks struct {
	Data []DaySchoolMarks
}

// DaySchoolMarks struct содержит в себе оценки и ДЗ на текущий день.
type DaySchoolMarks struct {
	Date    string
	Lessons []SchoolMark
}

// SchoolMark struct содержит в себе оценку и ДЗ по одному уроку.
type SchoolMark struct {
	AID    int
	CID    int
	TP     int
	Status bool
	InTime bool
	Name   string
	Author string
	Title  string
	Type   string
	Mark   string
	Weight string
}

// GetWeekSchoolMarks возвращает оценки на текущую неделю.
func (s *Session) GetWeekSchoolMarks(date string) (*WeekSchoolMarks, error) {
	var err error
	var weekSchoolMarks *WeekSchoolMarks
	switch s.Serv.Type {
	case cp.FirstType:
		weekSchoolMarks, err = s.getSchoolMarksFirst(date)
	default:
		err = fmt.Errorf("Unknown SchoolServer Type: %d", s.Serv.Type)
	}
	return weekSchoolMarks, err
}

/*
Получение отчетов.
*/

// TotalMarkReport struct - отчет первого типа.
type TotalMarkReport struct {
	Data map[string][]int
}

// GetTotalMarkReport возвращает "Отчет об успеваемости".
func (s *Session) GetTotalMarkReport() (*TotalMarkReport, error) {
	var err error
	var finalMarkReport *TotalMarkReport
	switch s.Serv.Type {
	case cp.FirstType:
		finalMarkReport, err = s.getTotalMarkReportFirst()
	default:
		err = fmt.Errorf("Unknown SchoolServer Type: %d", s.Serv.Type)
	}
	return finalMarkReport, err
}
