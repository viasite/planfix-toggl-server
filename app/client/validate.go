package client

import (
	"fmt"
	"github.com/viasite/planfix-toggl-server/app/config"
)

// Validator - интерфейс структуры валидатора
type Validator interface {
	Check() (errors []string, ok bool, data interface{})
}

// ValidatorStatus - статус ответа на запрос валидации
type ValidatorStatus struct {
	Name   string      `json:"name"`
	Ok     bool        `json:"ok"`
	Errors []string    `json:"errors"`
	Data   interface{} `json:"data"`
}

func StatusFromCheck(errors []string, ok bool, data interface{}) ValidatorStatus {
	return ValidatorStatus{
		Errors: errors,
		Ok:     ok,
		Data:   data,
	}
}

// errorToStrings помогает превратить ошибку в массив строк
func errorToStrings(err error, msg string) (errors []string) {
	if err != nil {
		errors = []string{fmt.Sprintf(msg, err.Error())}
	}
	return
}

// TogglUserValidator проверяет логин в Toggl
type TogglUserValidator struct {
	TogglClient *TogglClient
}

func (v TogglUserValidator) Check() (errors []string, ok bool, data interface{}) {
	user, err := v.TogglClient.GetTogglUser()
	errors = errorToStrings(err, "Не удалось получить Planfix UserID, проверьте PlanfixAPIKey, PlanfixAPIUrl, PlanfixUserName, PlanfixUserPassword, %s")
	return errors, err == nil, user
}

// TogglWorkspaceValidator проверяет workspace в Toggl
type TogglWorkspaceValidator struct {
	TogglClient *TogglClient
}

func (v TogglWorkspaceValidator) Check() (errors []string, ok bool, data interface{}) {
	ok, err := v.TogglClient.IsWorkspaceExists(v.TogglClient.Config.TogglWorkspaceID)
	errors = errorToStrings(err, "%s")
	if !ok {
		errors = append(errors, fmt.Sprintf("Toggl workspace ID %d не найден", v.TogglClient.Config.TogglWorkspaceID))
	}
	return errors, ok, v.TogglClient.Config.TogglWorkspaceID
}

// PlanfixUserValidator проверяет логин в Планфикс
type PlanfixUserValidator struct {
	TogglClient *TogglClient
}

func (v PlanfixUserValidator) Check() (errors []string, ok bool, data interface{}) {
	user, err := v.TogglClient.PlanfixAPI.UserGet(0)
	errors = errorToStrings(err, "Не удалось получить Planfix UserID, проверьте PlanfixAPIKey, PlanfixAPIUrl, PlanfixUserName, PlanfixUserPassword, %s")
	return errors, err == nil, user.User
}

// PlanfixAnaliticValidator проверяет аналитику
type PlanfixAnaliticValidator struct {
	TogglClient *TogglClient
}

func (v PlanfixAnaliticValidator) Check() (errors []string, ok bool, data interface{}) {
	analitic, err := v.TogglClient.GetAnaliticData(
		v.TogglClient.Config.PlanfixAnaliticName,
		v.TogglClient.Config.PlanfixAnaliticTypeName,
		v.TogglClient.Config.PlanfixAnaliticTypeValue,
		v.TogglClient.Config.PlanfixAnaliticCountName,
		v.TogglClient.Config.PlanfixAnaliticCommentName,
		v.TogglClient.Config.PlanfixAnaliticDateName,
		v.TogglClient.Config.PlanfixAnaliticUsersName,
	)
	errors = errorToStrings(err, "Поля аналитики указаны неправильно: %s")
	return errors, err == nil, analitic
}

// ConfigValidator проверяет конфиг на пустые или невалидные поля
type ConfigValidator struct {
	Config *config.Config
}

func (v ConfigValidator) Check() (errors []string, ok bool, config interface{}) {
	errors, ok = v.Config.Validate()
	config = v.Config
	return
}
