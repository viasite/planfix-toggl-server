package client

import (
	"fmt"
	"github.com/popstas/go-toggl"
	"github.com/popstas/planfix-go/planfix"
	"github.com/viasite/planfix-toggl-server/app/config"
	"log"
	"math"
	"net/smtp"
	"regexp"
	"strconv"
	"time"
)

// данные не меняются при этой опции
var testMode = false
var analiticDataCached PlanfixAnaliticData

// TogglClient - Клиент, общающийся с Toggl и Планфиксом
type TogglClient struct {
	Session    toggl.Session
	Config     config.Config
	PlanfixAPI planfix.API
	Logger     *log.Logger
}

// PlanfixEntryData - Данные, подмешивающиеся к toggl.DetailedTimeEntry
type PlanfixEntryData struct {
	Sent       bool `json:"sent"`
	TaskID     int  `json:"task_id"`
	GroupCount int  `json:"group_count"`
}

// TogglPlanfixEntry - toggl.DetailedTimeEntry дополнительными данными о задаче в Планфиксе
type TogglPlanfixEntry struct {
	toggl.DetailedTimeEntry
	Planfix PlanfixEntryData `json:"planfix"`
}

// TogglPlanfixEntryGroup - группа toggl-записей, объединенных одной задачей Планфикса
type TogglPlanfixEntryGroup struct {
	Entries         []TogglPlanfixEntry
	Description     string
	Project         string
	ProjectHexColor string
	Duration        int64
}

// PlanfixAnaliticData - данные аналитики, которая будет проставляться в Планфикс
type PlanfixAnaliticData struct {
	ID          int
	TypeID      int
	TypeValueID int
	CountID     int
	CommentID   int
	DateID      int
	UsersID     int
}

// RunSender - запускалка цикла отправки накопившихся toggl-записей
func (c TogglClient) RunSender() {
	time.Sleep(1 * time.Second) // wait for server start
	for {
		c.SendToPlanfix()
		time.Sleep(time.Duration(c.Config.SendInterval) * time.Minute)
	}
	tick := time.Tick(5 * time.Second)
	for _ = range tick {
		c.SendToPlanfix()
	}
}

// RunTagCleaner - запускалка цикла очистки запущенных toggl-записей от тега sent
func (c TogglClient) RunTagCleaner() {
	time.Sleep(1 * time.Second) // wait for server start
	for {
		entry, err := c.Session.GetCurrentTimeEntry()
		if err != nil {
			c.Logger.Println("[ERROR] failed to get current toggl entry")
			continue
		}

		// delete sent tag
		for _, tag := range entry.Tags {
			if tag == c.Config.TogglSentTag {
				c.Logger.Printf("[INFO] removed %s tag from current toggl entry", c.Config.TogglSentTag)
				c.Session.AddRemoveTag(entry.ID, c.Config.TogglSentTag, false)
			}
		}

		time.Sleep(1 * time.Minute)
	}
}

// SendToPlanfix получает записи из Toggl и отправляет в Планфикс
// * нужна, чтобы сохранился c.PlanfixAPI.Sid при авторизации
func (c *TogglClient) SendToPlanfix() (sumEntries []TogglPlanfixEntry, err error) {
	c.Logger.Println("[INFO] send to planfix")
	pendingEntries, err := c.GetPendingEntries()
	if err != nil {
		return []TogglPlanfixEntry{}, err
	}
	c.Logger.Printf("[INFO] found %d pending entries", len(pendingEntries))
	grouped := c.GroupEntriesByTask(pendingEntries)
	for taskID, entries := range grouped {
		err := c.sendEntries(taskID, entries)
		if err != nil {
			c.Logger.Printf("[ERROR] entries of task #%d failed to send", taskID)
		} else {
			c.Logger.Printf("[INFO] entries sent to https://%s.planfix.ru/task/%d", c.Config.PlanfixAccount, taskID)
		}
	}
	return c.SumEntriesGroup(grouped), nil
}

// SumEntriesGroup объединяет несколько toggl-записей в одну с просуммированным временем
// входной map формируется через GroupEntriesByTask
func (c TogglClient) SumEntriesGroup(grouped map[int][]TogglPlanfixEntry) (summed []TogglPlanfixEntry) {
	g := make(map[int]TogglPlanfixEntry)
	for taskID, entries := range grouped {
		for _, entry := range entries {
			if ge, ok := g[taskID]; ok {
				ge.Duration += entry.Duration
				ge.Planfix.GroupCount += 1
				g[entry.Planfix.TaskID] = ge
			} else {
				g[entry.Planfix.TaskID] = entry
			}
		}
	}

	summed = make([]TogglPlanfixEntry, 0, len(g))
	for _, entry := range g {
		summed = append(summed, entry)
	}
	return summed
}

// GroupEntriesByTask объединяет плоский список toggl-записей в map c ключом - ID задачи в Планфиксе
func (c TogglClient) GroupEntriesByTask(entries []TogglPlanfixEntry) (grouped map[int][]TogglPlanfixEntry) {
	grouped = make(map[int][]TogglPlanfixEntry)
	if len(entries) == 0 {
		return grouped
	}
	for _, entry := range entries {
		grouped[entry.Planfix.TaskID] = append(grouped[entry.Planfix.TaskID], entry)
	}
	return grouped
}

// GetUserData возвращает аккаунт юзера в Toggl
func (c TogglClient) GetUserData() (account toggl.Account) {
	account, err := c.Session.GetAccount()
	if err != nil {
		println("error:", err)
	}
	return account
}

// GetEntries получает []toggl.DetailedTimeEntry и превращает их в []TogglPlanfixEntry с подмешенными данными Планфикса
func (c TogglClient) GetEntries(togglWorkspaceID int, since, until string) (entries []TogglPlanfixEntry, err error) {
	report, err := c.Session.GetDetailedReport(togglWorkspaceID, since, until, 1)
	if err != nil {
		c.Logger.Printf("[ERROR] Toggl: %s", err)
	}

	for _, entry := range report.Data {

		pfe := TogglPlanfixEntry{
			DetailedTimeEntry: entry,
			Planfix: PlanfixEntryData{
				Sent:       false,
				TaskID:     0,
				GroupCount: 1,
			},
		}

		for _, tag := range entry.Tags {
			// only digit == planfix.task_id
			regex := regexp.MustCompile(`^\d+$`)
			if regex.MatchString(tag) {
				pfe.Planfix.TaskID, _ = strconv.Atoi(tag)
			}

			// sent tag
			if tag == c.Config.TogglSentTag {
				pfe.Planfix.Sent = true
			}
		}

		entries = append(entries, pfe)
	}

	return entries, nil
}

// filter - хэлпер, фильтрующий toggl-записи
func filter(input []TogglPlanfixEntry, f func(entry TogglPlanfixEntry) bool) (output []TogglPlanfixEntry) {
	for _, v := range input {
		if f(v) {
			output = append(output, v)
		}
	}
	return output
}

// GetPendingEntries возвращает toggl-записи, которые должны быть отправлены в Планфикс
func (c TogglClient) GetPendingEntries() ([]TogglPlanfixEntry, error) {
	user := c.GetUserData()
	entries, err := c.GetEntries(
		c.Config.TogglWorkspaceID,
		time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
		time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
	)
	if err != nil {
		return []TogglPlanfixEntry{}, err
	}
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return entry.Planfix.TaskID != 0 })
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return !entry.Planfix.Sent })
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return entry.Uid == user.Data.ID })
	return entries, nil
}

// sendEntries отправляет toggl-записи в Планфикс и помечает их в Toggl тегом sent
func (c TogglClient) sendEntries(planfixTaskID int, entries []TogglPlanfixEntry) error {
	var sumDuration int64
	for _, entry := range entries {
		sumDuration = sumDuration + entry.Duration
	}
	mins := int(math.Floor(float64(sumDuration)/60000 + .5))

	firstEntry := entries[0]

	entryString := fmt.Sprintf(
		"[%s] %s (%d)",
		firstEntry.Project,
		firstEntry.Description,
		mins,
	)
	c.Logger.Printf("[DEBUG] sending %s", entryString)

	date := firstEntry.Start.Format("2006-01-02")
	comment := fmt.Sprintf(
		"toggl: [%s] %s",
		firstEntry.Project,
		firstEntry.Description,
	)

	if testMode {
		return nil
	}

	// send to planfix
	var err error
	if c.Config.PlanfixUserName != "" && c.Config.PlanfixUserPassword != "" {
		err = c.sendWithPlanfixAPI(planfixTaskID, date, mins, comment)
	} else {
		err = c.sendWithSMTP(planfixTaskID, date, mins)
	}
	if err != nil {
		c.Logger.Printf("[ERROR] %v", err)
		return err
	}

	// mark as sent in toggl
	for _, entry := range entries {
		entryString := fmt.Sprintf(
			"[%s] %s (%d)",
			entry.Project,
			entry.Description,
			int(math.Floor(float64(entry.Duration)/60000+.5)),
		)
		c.Logger.Printf("[DEBUG] marking %s in toggl", entryString)
		if _, err := c.Session.AddRemoveTag(entry.ID, c.Config.TogglSentTag, true); err != nil {
			c.Logger.Println(err)
			return err
		}
	}
	return nil
}

// sendWithSMTP отправляет toggl-запись через SMTP
func (c TogglClient) sendWithSMTP(planfixTaskID int, date string, mins int) error {
	auth := smtp.PlainAuth("", c.Config.SMTPLogin, c.Config.SMTPPassword, c.Config.SMTPHost)
	taskEmail := fmt.Sprintf("task+%d@%s.planfix.ru", planfixTaskID, c.Config.PlanfixAccount)
	testEmail := c.Config.SMTPEmailFrom
	//test2Email := "task+530436@tagilcity.planfix.ru"
	to := []string{taskEmail, testEmail}
	body := fmt.Sprintf(
		"Content-Type: text/plain; charset=\"utf-8\"\r\n"+
			"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: @toggl @nonotify\r\n"+
			"\r\n"+
			"Вид работы: %s\r\n"+
			"time: %d\r\n"+
			"Автор: %s\r\n"+
			"Дата: %s\r\n",
		c.Config.SMTPEmailFrom,
		taskEmail,
		c.Config.PlanfixAnaliticTypeValue,
		mins,
		c.Config.PlanfixAuthorName,
		date,
	)
	msg := []byte(body)
	return smtp.SendMail(fmt.Sprintf("%s:%d", c.Config.SMTPHost, c.Config.SMTPPort), auth, c.Config.SMTPEmailFrom, to, msg)
}

// sendWithPlanfixAPI отправляет toggl-запись через Планфикс API
func (c TogglClient) sendWithPlanfixAPI(planfixTaskID int, date string, mins int, comment string) error {
	analiticData, err := c.getAnaliticData(
		c.Config.PlanfixAnaliticName,
		c.Config.PlanfixAnaliticTypeName,
		c.Config.PlanfixAnaliticCountName,
		c.Config.PlanfixAnaliticCommentName,
		c.Config.PlanfixAnaliticDateName,
		c.Config.PlanfixAnaliticUsersName,
	)
	if err != nil {
		return err
	}
	userIDs := struct {
		ID []int `xml:"id"`
	}{[]int{c.Config.PlanfixUserID}}

	_, err = c.PlanfixAPI.ActionAdd(planfix.XmlRequestActionAdd{
		TaskGeneral: planfixTaskID,
		Description: "",
		Analitics: []planfix.XmlRequestAnalitic{
			{
				ID: analiticData.ID,
				// аналитика должна содержать поля: вид работы, кол-во, дата, коммент, юзеры
				ItemData: []planfix.XmlRequestAnaliticField{
					{FieldID: analiticData.TypeID, Value: analiticData.TypeValueID}, // name
					{FieldID: analiticData.CountID, Value: mins},                    // count, минут
					{FieldID: analiticData.CommentID, Value: comment},               // comment
					{FieldID: analiticData.DateID, Value: date},                     // date
					{FieldID: analiticData.UsersID, Value: userIDs},                 // user
				},
			},
		},
	})
	return err
}

// getAnaliticData получает ID аналитики и ее полей из названий аналитики и полей
func (c TogglClient) getAnaliticData(name, typeName, countName, commentName, dateName, usersName string) (PlanfixAnaliticData, error) {
	if analiticDataCached.ID != 0 { // only on first call
		return analiticDataCached, nil
	}

	analitic, err := c.PlanfixAPI.GetAnaliticByName(name)
	if err != nil {
		return PlanfixAnaliticData{}, err
	}
	analiticOptions, err := c.PlanfixAPI.AnaliticGetOptions(analitic.ID)
	if err != nil {
		return PlanfixAnaliticData{}, err
	}

	analiticData := PlanfixAnaliticData{
		ID: analitic.ID,
	}

	for _, field := range analiticOptions.Analitic.Fields {
		if field.Name == typeName {
			analiticData.TypeID = field.ID
			record, err := c.PlanfixAPI.GetHandbookRecordByName(field.HandbookID, c.Config.PlanfixAnaliticTypeValue)
			if err != nil {
				return analiticData, err
			}
			analiticData.TypeValueID = record.Key
		}
		if field.Name == countName {
			analiticData.CountID = field.ID
		}
		if field.Name == commentName {
			analiticData.CommentID = field.ID
		}
		if field.Name == dateName {
			analiticData.DateID = field.ID
		}
		if field.Name == usersName {
			analiticData.UsersID = field.ID
		}
	}

	analiticDataCached = analiticData
	return analiticData, nil
}
