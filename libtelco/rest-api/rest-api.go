// rest-api.go

/*
Package restapi содержит handler'ы для взаимодействия сервера с клиентами.
*/
package restapi

import (
	cp "SchoolServer/libtelco/config-parser"
	"SchoolServer/libtelco/log"
	ss "SchoolServer/libtelco/sessions"
	db "SchoolServer/libtelco/sql-db"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
)

// RestAPI struct содержит конфигурацию Rest API.
// sessionsMap содержит отображения идентификаторов сессий Rest API
// в объекты сессий на удаленном сервере.
type RestAPI struct {
	config      *cp.Config
	store       *sessions.CookieStore
	logger      *log.Logger
	sessionsMap map[string]*ss.Session
	db          *db.Database
}

// errLoggedOut представляет объект ошибки когда клиент пытается выполнить запрос,
// а удаленная сессия была прервана
var errLoggedOut = fmt.Errorf("You was logged out from server")

// NewRestAPI создает структуру для работы с Rest API.
func NewRestAPI(logger *log.Logger, config *cp.Config) *RestAPI {
	key := make([]byte, 32)
	rand.Read(key)
	logger.Info("Generated secure key: ", key)
	newStore := sessions.NewCookieStore(key)
	newStore.MaxAge(86400 * 365)
	database, err := db.NewDatabase(logger)
	if err != nil {
		logger.Error("Error when creating database!")
	}
	return &RestAPI{
		config:      config,
		store:       newStore,
		logger:      logger,
		sessionsMap: make(map[string]*ss.Session),
		db:          database,
	}
}

// BindHandlers привязывает все handler'ы (с помощью http.HandleFunc).
func (rest *RestAPI) BindHandlers() {
	http.HandleFunc("/", rest.ErrorHandler)

	http.HandleFunc("/get_school_list", rest.GetSchoolListHandler)    // done
	http.HandleFunc("/check_permission", rest.CheckPermissionHandler) // done
	http.HandleFunc("/sign_in", rest.SignInHandler)                   // done
	http.HandleFunc("/log_out", rest.LogOutHandler)                   // done

	http.HandleFunc("/get_children_map", rest.GetChildrenMapHandler)      // done
	http.HandleFunc("/get_tasks_and_marks", rest.GetTasksAndMarksHandler) // done
	http.HandleFunc("/get_lesson_description", rest.Handler)
	http.HandleFunc("/mark_as_done", rest.Handler)
	http.HandleFunc("/unmark_as_done", rest.Handler)

	http.HandleFunc("/get_posts", rest.Handler)

	http.HandleFunc("/get_schedule", rest.GetScheduleHandler) // done

	http.HandleFunc("/get_report_student_total_marks", rest.GetReportStudentTotalMarksHandler)              // done
	http.HandleFunc("/get_report_student_average_mark", rest.GetReportStudentAverageMarkHandler)            // done
	http.HandleFunc("/get_report_student_average_mark_dyn", rest.GetReportStudentAverageMarkDynHandler)     // done
	http.HandleFunc("/get_report_student_grades_lesson_list", rest.GetReportStudentGradesLessonListHandler) // done
	http.HandleFunc("/get_report_student_grades", rest.Handler)
	http.HandleFunc("/get_report_student_total", rest.GetReportStudentTotalHandler) // done
	http.HandleFunc("/get_report_journal_access_classes_list", rest.Handler)
	http.HandleFunc("/get_report_journal_access", rest.Handler)
	http.HandleFunc("/get_report_parent_info_letter_data", rest.Handler)
	http.HandleFunc("/get_report_parent_info_letter", rest.GetReportParentInfoLetterHandler) // done

	http.HandleFunc("/get_resources", rest.Handler)

	http.HandleFunc("/get_mail", rest.Handler)
	http.HandleFunc("/get_mail_description", rest.Handler)
	http.HandleFunc("/delete_mail", rest.Handler)
	http.HandleFunc("/send_letter", rest.Handler)
	http.HandleFunc("/get_address_book", rest.Handler)

	http.HandleFunc("/get_forum", rest.Handler)
	http.HandleFunc("/get_forum_messages", rest.Handler)
	http.HandleFunc("/create_topic", rest.Handler)
	http.HandleFunc("/create_message_in_topic", rest.Handler)

	http.HandleFunc("/change_password", rest.Handler)
}

// checkPermissionRequest используется в CheckPermissionHandler
type checkPermissionRequest struct {
	Login string `json:"login"`
	ID    int    `json:"id"`
}

// checkPermissionResponse используется в CheckPermissionHandler
type checkPermissionResponse struct {
	Permission bool `json:"permission"`
}

// CheckPermissionHandler проверяет, есть ли разрешение на работу с школой
func (rest *RestAPI) CheckPermissionHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("CheckPermissionHandler called")
	if req.Method != "POST" {
		rest.logger.Error("Wrong method: ", req.Method)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Чтение json'a
	var rReq checkPermissionRequest
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&rReq)
	if err != nil {
		rest.logger.Error("Malformed request data")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Проверим разрешение у школы
	perm, err := rest.db.GetSchoolPermission(rReq.ID)
	if err != nil {
		rest.logger.Error("Invalid id param specified")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	if !perm {
		// Если у школы нет разрешения, проверить разрешение пользователя
		userPerm, err := rest.db.GetUserPermission(rReq.Login)
		if err != nil {
			if err.Error() == "record not found" {
				// Пользователь новый, вернем true
				perm = true
			} else {
				rest.logger.Error("Getting permission from db: ", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		perm = userPerm
	}
	resp := checkPermissionResponse{perm}
	bytes, err := json.Marshal(resp)
	if err != nil {
		rest.logger.Error("Error marshalling permission check response")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent permission check response: ", resp)
}

// ErrorHandler обрабатывает некорректные запросы.
func (rest *RestAPI) ErrorHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("Wrong request:", req.URL.EscapedPath())
}

// getReportStudentTotalMarksRequest используется в GetReportStudentTotalMarksHandler
type getReportStudentTotalMarksRequest struct {
	ID int `json:"id"`
}

// GetReportStudentTotalMarksHandler обрабатывает запрос на получение отчета
// об итоговых оценках
func (rest *RestAPI) GetReportStudentTotalMarksHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("GetReportStudentTotalMarksHandler called")
	// TODO добавить переключение между детьми
	if req.Method != "POST" {
		rest.logger.Error("Wrong method: ", req.Method)
		return
	}
	// Прочитать куку
	cookie, err := req.Cookie("sessionName")
	if err != nil {
		rest.logger.Info("User not authorized: sessionName absent")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionName := cookie.Value
	// Получить существующий объект сессии
	session, err := rest.store.Get(req, sessionName)
	if session.IsNew {
		rest.logger.Error("Local session broken")
		delete(rest.sessionsMap, sessionName)
		session.Options.MaxAge = -1
		session.Save(req, respwr)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Чтение запроса от клиента
	var rReq getReportStudentTotalMarksRequest
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&rReq)
	if err != nil {
		respwr.WriteHeader(http.StatusBadRequest)
		rest.logger.Error("Malformed request data")
		return
	}
	// Если нет удаленной сессии, создать
	remoteSession, ok := rest.sessionsMap[sessionName]
	if !ok {
		rest.logger.Info("No remote session, creating new one")
		userName := session.Values["userName"]
		school, err := rest.db.GetUserAuthData(userName.(string))
		if err != nil {
			rest.logger.Error("Error reading database")
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		remoteSession = ss.NewSession(school)
		if err = remoteSession.Login(); err != nil {
			rest.logger.Error("Error remote signing in")
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		rest.sessionsMap[sessionName] = remoteSession
	}
	id := strconv.Itoa(rReq.ID)
	totalMarkReport, err := remoteSession.GetTotalMarkReport(id)
	// Если удаленная сессия есть в mapSessions, но не активна, создать новую
	if err != nil {
		if err == errLoggedOut {
			rest.logger.Info("Remote connection broken, creation new one")
			userName := session.Values["userName"]
			school, err := rest.db.GetUserAuthData(userName.(string))
			if err != nil {
				rest.logger.Error("Error reading database")
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			remoteSession = ss.NewSession(school)
			if err = remoteSession.Login(); err != nil {
				rest.logger.Error("Error remote signing in")
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			rest.sessionsMap[sessionName] = remoteSession
			rest.logger.Info("Successfully created new remote session")
		} else {
			rest.logger.Error("Unable to get total marks: ", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	bytes, err := json.Marshal(totalMarkReport)
	if err != nil {
		rest.logger.Error("Error marshalling totalMarkReport")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent total marks report: ", totalMarkReport)
}

// getReportStudentAverageMarkRequest используется в GetReportStudentAverageMarkHandler
// и GetReportStudentAverageMarkDynHandler
type getReportStudentAverageMarkRequest struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
	From string `json:"from"`
	To   string `json:"to"`
}

// GetReportStudentAverageMarkHandler обрабатывает запрос на получение отчета
// о среднем балле
func (rest *RestAPI) GetReportStudentAverageMarkHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("GetReportStudentAverageMarkHandler called")
	if req.Method != "POST" {
		rest.logger.Error("Wrong method: ", req.Method)
		return
	}
	// Прочитать куку
	cookie, err := req.Cookie("sessionName")
	if err != nil {
		rest.logger.Info("User not authorized: sessionName absent")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionName := cookie.Value
	// Получить существующий объект сессии
	session, err := rest.store.Get(req, sessionName)
	if session.IsNew {
		rest.logger.Error("Local session broken")
		delete(rest.sessionsMap, sessionName)
		session.Options.MaxAge = -1
		session.Save(req, respwr)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Чтение запроса от клиента
	var rReq getReportStudentAverageMarkRequest
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&rReq)
	if err != nil {
		respwr.WriteHeader(http.StatusBadRequest)
		rest.logger.Error("Malformed request data")
		return
	}
	id := strconv.Itoa(rReq.ID)
	// Если нет удаленной сессии, создать
	remoteSession, ok := rest.sessionsMap[sessionName]
	if !ok {
		rest.logger.Info("No remote session, creating new one")
		userName := session.Values["userName"]
		school, err := rest.db.GetUserAuthData(userName.(string))
		if err != nil {
			rest.logger.Error("Error reading database", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		remoteSession = ss.NewSession(school)
		if err = remoteSession.Login(); err != nil {
			rest.logger.Error("Error remote signing in", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		rest.sessionsMap[sessionName] = remoteSession
	}

	averageMarkReport, err := remoteSession.GetAverageMarkReport(rReq.From, rReq.To, rReq.Type, id)
	// Если удаленная сессия есть в mapSessions, но не активна, создать новую
	if err != nil {
		if err == errLoggedOut {
			rest.logger.Info("Remote connection broken, creation new one")
			userName := session.Values["userName"]
			school, err := rest.db.GetUserAuthData(userName.(string))
			if err != nil {
				rest.logger.Error("Error reading database", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			remoteSession = ss.NewSession(school)
			if err = remoteSession.Login(); err != nil {
				rest.logger.Error("Error remote signing in", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			rest.sessionsMap[sessionName] = remoteSession
			rest.logger.Info("Successfully created new remote session")
		} else {
			rest.logger.Error("Unable to get average marks: ", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	bytes, err := json.Marshal(averageMarkReport)
	if err != nil {
		rest.logger.Error("Error marshalling averageMarkReport")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent average marks report: ", averageMarkReport)
}

// GetReportStudentAverageMarkDynHandler обрабатывает запрос на получение отчета
// о динамике среднего балла
func (rest *RestAPI) GetReportStudentAverageMarkDynHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("GetReportStudentAverageMarkDynHandler called")
	if req.Method != "POST" {
		rest.logger.Error("Wrong method: ", req.Method)
		return
	}
	// Прочитать куку
	cookie, err := req.Cookie("sessionName")
	if err != nil {
		rest.logger.Info("User not authorized: sessionName absent")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionName := cookie.Value
	// Получить существующий объект сессии
	session, err := rest.store.Get(req, sessionName)
	if session.IsNew {
		rest.logger.Error("Local session broken")
		delete(rest.sessionsMap, sessionName)
		session.Options.MaxAge = -1
		session.Save(req, respwr)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Чтение запроса от клиента
	var rReq getReportStudentAverageMarkRequest
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&rReq)
	if err != nil {
		respwr.WriteHeader(http.StatusBadRequest)
		rest.logger.Error("Malformed request data")
		return
	}
	id := strconv.Itoa(rReq.ID)
	// Если нет удаленной сессии, создать
	remoteSession, ok := rest.sessionsMap[sessionName]
	if !ok {
		rest.logger.Info("No remote session, creating new one")
		userName := session.Values["userName"]
		school, err := rest.db.GetUserAuthData(userName.(string))
		if err != nil {
			rest.logger.Error("Error reading database", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		remoteSession = ss.NewSession(school)
		if err = remoteSession.Login(); err != nil {
			rest.logger.Error("Error remote signing in", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		rest.sessionsMap[sessionName] = remoteSession
	}
	averageMarkDynReport, err := remoteSession.GetAverageMarkDynReport(rReq.From, rReq.To, rReq.Type, id)
	// Если удаленная сессия есть в mapSessions, но не активна, создать новую
	if err != nil {
		if err == errLoggedOut {
			rest.logger.Info("Remote connection broken, creation new one")
			userName := session.Values["userName"]
			school, err := rest.db.GetUserAuthData(userName.(string))
			if err != nil {
				rest.logger.Error("Error reading database", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			remoteSession = ss.NewSession(school)
			if err = remoteSession.Login(); err != nil {
				rest.logger.Error("Error remote signing in", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			rest.sessionsMap[sessionName] = remoteSession
			rest.logger.Info("Successfully created new remote session")
		} else {
			rest.logger.Error("Unable to get average dyn marks: ", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	bytes, err := json.Marshal(averageMarkDynReport)
	if err != nil {
		rest.logger.Error("Error marshalling averageMarkDynReport")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent average marks dynamic report: ", averageMarkDynReport)
}

// getReportStudentGradesLessonListRequest используется в GetReportStudentGradesLessonListHandler
type getReportStudentGradesLessonListRequest struct {
	ID int `json:"id"`
}

// GetReportStudentGradesLessonListHandler обрабатывает запрос на получение
// списка предметов для отчета 'Об успеваемости'
func (rest *RestAPI) GetReportStudentGradesLessonListHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("GetReportStudentGradesLessonListHandler called")
	if req.Method != "POST" {
		rest.logger.Error("Wrong method: ", req.Method)
		return
	}
	// Прочитать куку
	cookie, err := req.Cookie("sessionName")
	if err != nil {
		rest.logger.Info("User not authorized: sessionName absent")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionName := cookie.Value
	// Получить существующий объект сессии
	session, err := rest.store.Get(req, sessionName)
	if session.IsNew {
		rest.logger.Error("Local session broken")
		delete(rest.sessionsMap, sessionName)
		session.Options.MaxAge = -1
		session.Save(req, respwr)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Чтение запроса от клиента
	var rReq getReportStudentGradesLessonListRequest
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&rReq)
	if err != nil {
		respwr.WriteHeader(http.StatusBadRequest)
		rest.logger.Error("Malformed request data")
		return
	}
	id := strconv.Itoa(rReq.ID)
	// Если нет удаленной сессии, создать
	remoteSession, ok := rest.sessionsMap[sessionName]
	if !ok {
		rest.logger.Info("No remote session, creating new one")
		userName := session.Values["userName"]
		school, err := rest.db.GetUserAuthData(userName.(string))
		if err != nil {
			rest.logger.Error("Error reading database", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		remoteSession = ss.NewSession(school)
		if err = remoteSession.Login(); err != nil {
			rest.logger.Error("Error remote signing in", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		rest.sessionsMap[sessionName] = remoteSession
	}
	lessonsMap, err := remoteSession.GetLessonsMap(id)
	// Если удаленная сессия есть в mapSessions, но не активна, создать новую
	if err != nil {
		if err == errLoggedOut {
			rest.logger.Info("Remote connection broken, creation new one")
			userName := session.Values["userName"]
			school, err := rest.db.GetUserAuthData(userName.(string))
			if err != nil {
				rest.logger.Error("Error reading database", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			remoteSession = ss.NewSession(school)
			if err = remoteSession.Login(); err != nil {
				rest.logger.Error("Error remote signing in", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			rest.sessionsMap[sessionName] = remoteSession
			rest.logger.Info("Successfully created new remote session")
		} else {
			rest.logger.Error("Unable to get lessons map: ", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	bytes, err := json.Marshal(lessonsMap)
	if err != nil {
		rest.logger.Error("Error marshalling lessonsMap")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent report student grades lesson: ", lessonsMap)
}

// getReportStudentTotalRequest используется в GetReportStudentTotalHandler
type getReportStudentTotalRequest struct {
	ID   int    `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

// GetReportStudentTotalHandler брабатывает запрос на получение отчета об успеваемости
// и посещаемости
func (rest *RestAPI) GetReportStudentTotalHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("GetReportStudentTotalHandler called")
	if req.Method != "POST" {
		rest.logger.Error("Wrong method: ", req.Method)
		return
	}
	// Прочитать куку
	cookie, err := req.Cookie("sessionName")
	if err != nil {
		rest.logger.Info("User not authorized: sessionName absent")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionName := cookie.Value
	// Получить существующий объект сессии
	session, err := rest.store.Get(req, sessionName)
	if session.IsNew {
		rest.logger.Error("Local session broken")
		delete(rest.sessionsMap, sessionName)
		session.Options.MaxAge = -1
		session.Save(req, respwr)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Чтение запроса от клиента
	var rReq getReportStudentTotalRequest
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&rReq)
	if err != nil {
		respwr.WriteHeader(http.StatusBadRequest)
		rest.logger.Error("Malformed request data")
		return
	}
	// Если нет удаленной сессии, создать
	remoteSession, ok := rest.sessionsMap[sessionName]
	if !ok {
		rest.logger.Info("No remote session, creating new one")
		userName := session.Values["userName"]
		school, err := rest.db.GetUserAuthData(userName.(string))
		if err != nil {
			rest.logger.Error("Error reading database", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		remoteSession = ss.NewSession(school)
		if err = remoteSession.Login(); err != nil {
			rest.logger.Error("Error remote signing in", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		rest.sessionsMap[sessionName] = remoteSession
	}
	studentID := strconv.Itoa(rReq.ID)
	totalReport, err := remoteSession.GetStudentTotalReport(rReq.From, rReq.To, studentID)
	// Если удаленная сессия есть в mapSessions, но не активна, создать новую
	if err != nil {
		if err == errLoggedOut {
			rest.logger.Info("Remote connection broken, creation new one")
			userName := session.Values["userName"]
			school, err := rest.db.GetUserAuthData(userName.(string))
			if err != nil {
				rest.logger.Error("Error reading database", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			remoteSession = ss.NewSession(school)
			if err = remoteSession.Login(); err != nil {
				rest.logger.Error("Error remote signing in", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			rest.sessionsMap[sessionName] = remoteSession
			rest.logger.Info("Successfully created new remote session")
		} else {
			rest.logger.Error("Unable to get total student report: ", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	bytes, err := json.Marshal(totalReport)
	if err != nil {
		rest.logger.Error("Error marshalling totalReport")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent total student report: ", totalReport)
}

// getReportParentInfoLetterRequest используется в GetReportParentInfoLetterHandler
type getReportParentInfoLetterRequest struct {
	StudentID    int `json:"student_id"`
	ReportTypeID int `json:"report_type_id"`
	PeriodID     int `json:"period_id"`
}

// GetReportParentInfoLetterHandler обрабатывает запрос на получение шаблона для
// письма родителям
func (rest *RestAPI) GetReportParentInfoLetterHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("GetReportParentInfoLetterHandler called")
	if req.Method != "POST" {
		rest.logger.Error("Wrong method: ", req.Method)
		return
	}
	// Прочитать куку
	cookie, err := req.Cookie("sessionName")
	if err != nil {
		rest.logger.Info("User not authorized: sessionName absent")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionName := cookie.Value
	// Получить существующий объект сессии
	session, err := rest.store.Get(req, sessionName)
	if session.IsNew {
		rest.logger.Error("Local session broken")
		delete(rest.sessionsMap, sessionName)
		session.Options.MaxAge = -1
		session.Save(req, respwr)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Чтение запроса от клиента
	var rReq getReportParentInfoLetterRequest
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&rReq)
	if err != nil {
		respwr.WriteHeader(http.StatusBadRequest)
		rest.logger.Error("Malformed request data")
		return
	}
	// Если нет удаленной сессии, создать
	remoteSession, ok := rest.sessionsMap[sessionName]
	if !ok {
		rest.logger.Info("No remote session, creating new one")
		userName := session.Values["userName"]
		school, err := rest.db.GetUserAuthData(userName.(string))
		if err != nil {
			rest.logger.Error("Error reading database", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		remoteSession = ss.NewSession(school)
		if err = remoteSession.Login(); err != nil {
			rest.logger.Error("Error remote signing in", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		rest.sessionsMap[sessionName] = remoteSession
	}
	studentID := strconv.Itoa(rReq.StudentID)
	reportID := strconv.Itoa(rReq.ReportTypeID)
	periodID := strconv.Itoa(rReq.PeriodID)
	parentLetter, err := remoteSession.GetParentInfoLetterReport(studentID, reportID, periodID)
	// Если удаленная сессия есть в mapSessions, но не активна, создать новую
	if err != nil {
		if err == errLoggedOut {
			rest.logger.Info("Remote connection broken, creation new one")
			userName := session.Values["userName"]
			school, err := rest.db.GetUserAuthData(userName.(string))
			if err != nil {
				rest.logger.Error("Error reading database", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			remoteSession = ss.NewSession(school)
			if err = remoteSession.Login(); err != nil {
				rest.logger.Error("Error remote signing in", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			rest.sessionsMap[sessionName] = remoteSession
			rest.logger.Info("Successfully created new remote session")
		} else {
			rest.logger.Error("Unable to get parent info letter report: ", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	bytes, err := json.Marshal(parentLetter)
	if err != nil {
		rest.logger.Error("Error marshalling parentLetter")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent parent info letter report: ", parentLetter)
}

// school struct используется в GetSchoolListHandler
type school struct {
	Name    string `json:"name"`
	ID      int    `json:"id"`
	Website string `json:"website"`
}

// SchoolListResponse используется в GetSchoolListHandler
type SchoolListResponse struct {
	Schools []school `json:"schools"`
}

// GetSchoolListHandler обрабатывает запрос на получение списка обслуживаемых школ
func (rest *RestAPI) GetSchoolListHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("GetSchoolListHandler called")
	if req.Method != "GET" {
		rest.logger.Error("Wrong method: ", req.Method)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	schools, err := rest.db.GetSchools()
	if err != nil {
		rest.logger.Error("Error getting school list from db")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	schoolList := make([]school, 0)
	for _, sch := range schools {
		schoolList = append(schoolList, school{sch.Name, int(sch.ID), sch.Address})
	}
	resp := SchoolListResponse{schoolList}
	bytes, err := json.Marshal(resp)
	if err != nil {
		rest.logger.Error("Error marshalling list of schools")
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent list of schools: ", resp)
}

// GetChildrenMapHandler обрабатывает запрос на получение списка детей
func (rest *RestAPI) GetChildrenMapHandler(respwr http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		rest.logger.Error("Wrong method: ", req.Method)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Прочитать куку
	cookie, err := req.Cookie("sessionName")
	if err != nil {
		rest.logger.Info("User not authorized: sessionName absent")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionName := cookie.Value
	// Получить существующий объект сессии
	session, err := rest.store.Get(req, sessionName)
	if session.IsNew {
		rest.logger.Error("Local session broken")
		delete(rest.sessionsMap, sessionName)
		session.Options.MaxAge = -1
		session.Save(req, respwr)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Если нет удаленной сессии, создать
	remoteSession, ok := rest.sessionsMap[sessionName]
	if !ok {
		rest.logger.Info("No remote session, creating new one")
		userName := session.Values["userName"]
		school, err := rest.db.GetUserAuthData(userName.(string))
		if err != nil {
			rest.logger.Error("Error reading database")
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		remoteSession = ss.NewSession(school)
		if err = remoteSession.Login(); err != nil {
			rest.logger.Error("Error remote signing in")
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		rest.sessionsMap[sessionName] = remoteSession
	}
	err = remoteSession.GetChildrenMap()
	// Если удаленная сессия есть в mapSessions, но не активна, создать новую
	if err != nil {
		if err == errLoggedOut {
			rest.logger.Info("Remote connection broken, creation new one")
			userName := session.Values["userName"]
			school, err := rest.db.GetUserAuthData(userName.(string))
			if err != nil {
				rest.logger.Error("Error reading database")
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			remoteSession = ss.NewSession(school)
			if err = remoteSession.Login(); err != nil {
				rest.logger.Error("Error remote signing in")
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			rest.sessionsMap[sessionName] = remoteSession
			rest.logger.Info("Successfully created new remote session")
		} else {
			rest.logger.Error("Unable to get children map: ", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	bytes, err := json.Marshal(remoteSession.Base.ChildrenIDS)
	if err != nil {
		rest.logger.Error("Error marshalling childrenMap")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent children map: ", remoteSession.Base.ChildrenIDS)
}

// tasksMarksRequest используется в GetTasksAndMarksHandler
type tasksMarksRequest struct {
	Week string `json:"week"`
	ID   int    `json:"id"`
}

// GetTasksAndMarksHandler возвращает задания и оценки на неделю
func (rest *RestAPI) GetTasksAndMarksHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("GetTasksAndMarksHandler called")
	if req.Method != "POST" {
		rest.logger.Error("Wrong method: ", req.Method)
		return
	}
	// Прочитать куку
	cookie, err := req.Cookie("sessionName")
	if err != nil {
		rest.logger.Info("User not authorized: sessionName absent")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionName := cookie.Value
	// Получить существующий объект сессии
	session, err := rest.store.Get(req, sessionName)
	if session.IsNew {
		rest.logger.Error("Local session broken")
		delete(rest.sessionsMap, sessionName)
		session.Options.MaxAge = -1
		session.Save(req, respwr)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Чтение запроса от клиента
	var rReq tasksMarksRequest
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&rReq)
	if err != nil {
		respwr.WriteHeader(http.StatusBadRequest)
		rest.logger.Error("Malformed request data")
		return
	}
	id := strconv.Itoa(rReq.ID)
	// Если нет удаленной сессии, создать
	remoteSession, ok := rest.sessionsMap[sessionName]
	if !ok {
		rest.logger.Info("No remote session, creating new one")
		userName := session.Values["userName"]
		school, err := rest.db.GetUserAuthData(userName.(string))
		if err != nil {
			rest.logger.Error("Error reading database")
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		remoteSession = ss.NewSession(school)
		if err = remoteSession.Login(); err != nil {
			rest.logger.Error("Error remote signing in")
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		rest.sessionsMap[sessionName] = remoteSession
	}
	week := rReq.Week
	if week == "" {
		week = time.Now().Format("02.01.2006")
	}
	weekMarks, err := remoteSession.GetWeekSchoolMarks(week, id)
	// Если удаленная сессия есть в mapSessions, но не активна, создать новую
	if err != nil {
		if err == errLoggedOut {
			rest.logger.Info("Remote connection broken, creation new one")
			userName := session.Values["userName"]
			school, err := rest.db.GetUserAuthData(userName.(string))
			if err != nil {
				rest.logger.Error("Error reading database")
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			remoteSession = ss.NewSession(school)
			if err = remoteSession.Login(); err != nil {
				rest.logger.Error("Error remote signing in")
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			rest.sessionsMap[sessionName] = remoteSession
			rest.logger.Info("Successfully created new remote session")
		} else {
			rest.logger.Error("Unable to get schedule: ", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	bytes, err := json.Marshal(weekMarks)
	if err != nil {
		rest.logger.Error("Error marshalling weekMarks")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent tasks and marks for a week: ", weekMarks)
}

// scheduleRequest используется в GetScheduleHandler
type scheduleRequest struct {
	Days int `json:"days"`
	ID   int `json:"id"`
}

// GetScheduleHandler возвращает расписание на неделю
func (rest *RestAPI) GetScheduleHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("GetScheduleHandler called")
	if req.Method != "POST" {
		rest.logger.Error("Wrong method: ", req.Method)
		return
	}
	// Прочитать куку
	cookie, err := req.Cookie("sessionName")
	if err != nil {
		rest.logger.Info("User not authorized: sessionName absent")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionName := cookie.Value
	// Получить существующий объект сессии
	session, err := rest.store.Get(req, sessionName)
	if session.IsNew {
		rest.logger.Error("Local session broken")
		delete(rest.sessionsMap, sessionName)
		session.Options.MaxAge = -1
		session.Save(req, respwr)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Чтение запроса от клиента
	var rReq scheduleRequest
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&rReq)
	if err != nil {
		respwr.WriteHeader(http.StatusBadRequest)
		rest.logger.Error("Malformed request data")
		return
	}
	id := strconv.Itoa(rReq.ID)
	// Если нет удаленной сессии, создать
	remoteSession, ok := rest.sessionsMap[sessionName]
	if !ok {
		rest.logger.Info("No remote session, creating new one")
		userName := session.Values["userName"]
		school, err := rest.db.GetUserAuthData(userName.(string))
		if err != nil {
			rest.logger.Error("Error reading database", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		remoteSession = ss.NewSession(school)
		if err = remoteSession.Login(); err != nil {
			rest.logger.Error("Error remote signing in", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
		rest.sessionsMap[sessionName] = remoteSession
	}
	today := time.Now().Format("02.01.2006")
	timeTable, err := remoteSession.GetTimeTable(today, rReq.Days, id)
	// Если удаленная сессия есть в mapSessions, но не активна, создать новую
	if err != nil {
		if err == errLoggedOut {
			rest.logger.Info("Remote connection broken, creation new one")
			userName := session.Values["userName"]
			school, err := rest.db.GetUserAuthData(userName.(string))
			if err != nil {
				rest.logger.Error("Error reading database", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			remoteSession = ss.NewSession(school)
			if err = remoteSession.Login(); err != nil {
				rest.logger.Error("Error remote signing in", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
			rest.sessionsMap[sessionName] = remoteSession
			rest.logger.Info("Successfully created new remote session")
		} else {
			rest.logger.Error("Unable to get schedule: ", err)
			respwr.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	bytes, err := json.Marshal(timeTable)
	if err != nil {
		rest.logger.Error("Error marshalling timeTable")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	respwr.Write(bytes)
	rest.logger.Info("Sent schedule for a week: ", timeTable)
}

// LogOutHandler обрабатывает удаление сессии клиента и отвязку устройства
func (rest *RestAPI) LogOutHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("LogOutHandler called")
	if req.Method != "GET" {
		rest.logger.Error("Wrong method: ", req.Method)
		return
	}
	// Прочитать куку
	cookie, err := req.Cookie("sessionName")
	if err != nil {
		rest.logger.Info("User not authorized: sessionName absent")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	sessionName := cookie.Value

	/* TODO
	не чистить sessionMap, сохранять в БД идентификатор последней сессии,
	чтобы можно было к случае logout+login восстановить удаленную сессию
	по имени пользователя
	*/

	// Если кука есть, удалить локальную и удаленную сессии
	session, err := rest.store.Get(req, sessionName)
	if err != nil {
		rest.logger.Info("Error getting session: ", sessionName)
		return
	}
	delete(rest.sessionsMap, sessionName)
	session.Options.MaxAge = -1
	session.Save(req, respwr)
	respwr.WriteHeader(http.StatusOK)
	rest.logger.Info("Successful logout for session ", sessionName)
}

// signInRequest используется в SignInHandler
type signInRequest struct {
	Login   string `json:"login"`
	Passkey string `json:"passkey"`
	ID      int    `json:"id"`
}

// SignInHandler обрабатывает вход в учетную запись на сайте школы
func (rest *RestAPI) SignInHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("SignInHandler called")
	if req.Method != "POST" {
		rest.logger.Error("Wrong method: ", req.Method)
		return
	}
	// Чтение запроса от клиента
	var rReq signInRequest
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&rReq)
	if err != nil {
		respwr.WriteHeader(http.StatusBadRequest)
		rest.logger.Error("Malformed request data")
		return
	}
	// Проверим разрешение у школы
	perm, err := rest.db.GetSchoolPermission(rReq.ID)
	if err != nil {
		rest.logger.Error("Invalid id param specified")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	if !perm {
		// Если у школы нет разрешения, проверить разрешение пользователя
		userPerm, err := rest.db.GetUserPermission(rReq.Login)
		if err != nil {
			if err.Error() == "record not found" {
				// Пользователь новый, вернем true
				perm = true
			} else {
				rest.logger.Error("Getting permission from db: ", err)
				respwr.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		perm = userPerm
	}
	if !perm {
		rest.logger.Info("Access to service denied!")
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	rest.logger.Info("Valid data:", rReq)
	school := rest.config.Schools[rReq.ID-1]
	school.Login = rReq.Login
	school.Password = rReq.Passkey
	// Создание удаленной сессии
	newRemoteSession := ss.NewSession(&school)
	if err = newRemoteSession.Login(); err != nil {
		rest.logger.Error("Error remote signing in", err)
		respwr.WriteHeader(http.StatusBadRequest)
		return
	}
	// Если удаленная авторизация прошла успешно, создать локальную сессию
	timeString := time.Now().String()
	hasher := md5.New()
	if _, err = hasher.Write([]byte(timeString)); err != nil {
		rest.logger.Error("Md5 hashing error: ", err)
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	newSessionName := hex.EncodeToString(hasher.Sum(nil))
	newLocalSession, err := rest.store.Get(req, newSessionName)
	if err != nil {
		rest.logger.Error("Error creating new local session")
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	// ... и привязать к ней удаленную сессию
	rest.sessionsMap[newSessionName] = newRemoteSession
	newLocalSession.Values["userName"] = rReq.Login
	newLocalSession.Save(req, respwr)
	// Устанавливаем в куки значение sessionName
	expiration := time.Now().Add(365 * 24 * time.Hour)
	cookie := http.Cookie{
		Name: "sessionName", Value: newSessionName, Expires: expiration,
	}
	http.SetCookie(respwr, &cookie)
	// Обновляем базу данных
	err = rest.db.UpdateUser(rReq.Login, rReq.Passkey, rReq.ID)
	if err != nil {
		rest.logger.Error("Error updating database: ", err)
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	rest.logger.Info("Successfully signed in as user: ", rReq.Login)
}

// Handler временный абстрактный handler для некоторых еще не реализованных
// обработчиков запросов.
func (rest *RestAPI) Handler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("Handler called (not implemented yet)", req.URL.EscapedPath())
}
