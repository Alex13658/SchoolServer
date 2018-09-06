// posts
package restapi

import (
	"encoding/json"
	"net/http"
)

// GetPostsHandler обрабатывает запрос на получение объявлений
func (rest *RestAPI) GetPostsHandler(respwr http.ResponseWriter, req *http.Request) {
	rest.logger.Info("REST: GetPostsHandler called", "IP", req.RemoteAddr)
	// Проверка метода запроса
	if req.Method != "GET" {
		rest.logger.Info("REST: Wrong method", "Method", req.Method, "IP", req.RemoteAddr)
		respwr.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Получить существующие имя и объект локальной сессии
	sessionName, session := rest.getLocalSession(respwr, req)
	if session == nil {
		return
	}
	// Получим удаленную сессию
	remoteSession, ok := rest.sessionsMap[sessionName]
	if !ok {
		// Если нет удаленной сессии
		rest.logger.Info("REST: No remote session", "IP", req.RemoteAddr)
		// Создать новую
		remoteSession = rest.remoteRelogin(respwr, req, session)
		if remoteSession == nil {
			return
		}
	}
	/*
		// Получить описание таска
		lessonDescription, err := remoteSession.GetPosts()
		if err != nil {
			if strings.Contains(err.Error(), "You was logged out from server") {
				// Если удаленная сессия есть, но не активна
				rest.logger.Info("REST: Remote connection timed out", "IP", req.RemoteAddr)
				// Создать новую
				remoteSession = rest.remoteRelogin(respwr, req, session)
				if remoteSession == nil {
					return
				}
				// Повторно получить с сайта школы
				lessonDescription, err = remoteSession.GetPosts()
				if err != nil {
					// Ошибка
					rest.logger.Error("REST: Error occured when getting data from site", "Error", err, "IP", req.RemoteAddr)
					respwr.WriteHeader(http.StatusBadGateway)
					return
				}
			} else {
				// Другая ошибка
				rest.logger.Error("REST: Error occured when getting data from site", "Error", err, "IP", req.RemoteAddr)
				respwr.WriteHeader(http.StatusBadGateway)
				return
			}
		}
	*/
	posts := make([]string, 0)
	// Закодировать ответ в JSON
	bytes, err := json.Marshal(posts)
	if err != nil {
		rest.logger.Error("REST: Error occured when marshalling response", "Error", err, "Response", posts, "IP", req.RemoteAddr)
		respwr.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Отправить ответ клиенту
	status, err := respwr.Write(bytes)
	if err != nil {
		rest.logger.Error("REST: Error occured when sending response", "Error", err, "Response", posts, "Status", status, "IP", req.RemoteAddr)
	} else {
		rest.logger.Info("REST: Successfully sent response", "Response", posts, "IP", req.RemoteAddr)
	}
}
