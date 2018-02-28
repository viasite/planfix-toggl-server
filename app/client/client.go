package client

import (
	"github.com/popstas/go-toggl"
	"fmt"
	"github.com/viasite/planfix-toggl-server/app/config"
	"math"
	"log"
	"regexp"
	"strconv"
	"net/smtp"
	"time"
	"github.com/popstas/planfix-go/planfix"
)

// данные не меняются при этой опции
var testMode = false
var analiticDataCached PlanfixAnaliticData

type TogglClient struct {
	Session    toggl.Session
	Config     config.Config
	PlanfixApi planfix.Api
	Logger     *log.Logger
}

type PlanfixEntryData struct {
	Sent       bool `json:"sent"`
	TaskId     int  `json:"task_id"`
	GroupCount int  `json:"group_count"`
}

type TogglPlanfixEntry struct {
	toggl.DetailedTimeEntry
	Planfix PlanfixEntryData `json:"planfix"`
}

type TogglPlanfixEntryGroup struct {
	Entries         []TogglPlanfixEntry
	Description     string
	Project         string
	ProjectHexColor string
	Duration        int64
}

type PlanfixAnaliticData struct {
	Id          int
	TypeId      int
	TypeValueId int
	CountId     int
	CommentId   int
	DateId      int
	UsersId     int
}

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

// получает записи из Toggl и отправляет в Планфикс
// * нужна, чтобы сохранился c.PlanfixApi.Sid при авторизации
func (c *TogglClient) SendToPlanfix() (sumEntries []TogglPlanfixEntry, err error) {
	c.Logger.Println("[INFO] send to planfix")
	pendingEntries, err := c.GetPendingEntries()
	if err != nil {
		return []TogglPlanfixEntry{}, err
	}
	c.Logger.Printf("[INFO] found %d pending entries", len(pendingEntries))
	grouped := c.GroupEntriesByTask(pendingEntries)
	for taskId, entries := range grouped {
		err := c.sendEntries(taskId, entries)
		if err != nil {
			c.Logger.Printf("[ERROR] entries of task #%d failed to send", taskId)
		} else {
			c.Logger.Printf("[INFO] entries sent to https://%s.planfix.ru/task/%d", c.Config.PlanfixAccount, taskId)
		}
	}
	return c.SumEntriesGroup(grouped), nil
}

func (c TogglClient) SumEntriesGroup(grouped map[int][]TogglPlanfixEntry) (summed []TogglPlanfixEntry) {
	g := make(map[int]TogglPlanfixEntry)
	for taskId, entries := range grouped {
		for _, entry := range entries {
			if ge, ok := g[taskId]; ok {
				ge.Duration += entry.Duration
				ge.Planfix.GroupCount += 1
				g[entry.Planfix.TaskId] = ge
			} else {
				g[entry.Planfix.TaskId] = entry
			}
		}
	}

	summed = make([]TogglPlanfixEntry, 0, len(g))
	for _, entry := range g {
		summed = append(summed, entry)
	}
	return summed
}

func (c TogglClient) GroupEntriesByTask(entries []TogglPlanfixEntry) (grouped map[int][]TogglPlanfixEntry) {
	grouped = make(map[int][]TogglPlanfixEntry)
	if len(entries) == 0 {
		return grouped
	}
	for _, entry := range entries {
		grouped[entry.Planfix.TaskId] = append(grouped[entry.Planfix.TaskId], entry)
	}
	return grouped
}

func (c TogglClient) GetUserData() (account toggl.Account) {
	account, err := c.Session.GetAccount()
	if err != nil {
		println("error:", err)
	}
	return account
}

// report entries with planfix data
func (c TogglClient) GetEntries(togglWorkspaceId int, since, until string) (entries []TogglPlanfixEntry, err error) {
	report, err := c.Session.GetDetailedReport(togglWorkspaceId, since, until, 1);
	if err != nil {
		c.Logger.Printf("[ERROR] Toggl: %s", err)
	}

	for _, entry := range report.Data {

		pfe := TogglPlanfixEntry{
			DetailedTimeEntry: entry,
			Planfix: PlanfixEntryData{
				Sent:       false,
				TaskId:     0,
				GroupCount: 1,
			},
		}

		for _, tag := range entry.Tags {
			// only digit == planfix.task_id
			regex := regexp.MustCompile(`^\d+$`)
			if regex.MatchString(tag) {
				pfe.Planfix.TaskId, _ = strconv.Atoi(tag)
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

func filter(input []TogglPlanfixEntry, f func(entry TogglPlanfixEntry) bool) (output []TogglPlanfixEntry) {
	for _, v := range input {
		if f(v) {
			output = append(output, v)
		}
	}
	return output
}

func (c TogglClient) GetPendingEntries() ([]TogglPlanfixEntry, error) {
	user := c.GetUserData()
	entries, err := c.GetEntries(
		c.Config.TogglWorkspaceId,
		time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
		time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
	)
	if err != nil {
		return []TogglPlanfixEntry{}, err
	}
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return entry.Planfix.TaskId != 0 })
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return !entry.Planfix.Sent })
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return entry.Uid == user.Data.ID })
	return entries, nil
}

// отправка письма и пометка тегом sent в Toggl
func (c TogglClient) sendEntries(planfixTaskId int, entries []TogglPlanfixEntry) (error) {
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
		err = c.sendWithPlanfixApi(planfixTaskId, date, mins, comment)
	} else {
		err = c.sendWithSmtp(planfixTaskId, date, mins)
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

func (c TogglClient) sendWithSmtp(planfixTaskId int, date string, mins int) error {
	auth := smtp.PlainAuth("", c.Config.SmtpLogin, c.Config.SmtpPassword, c.Config.SmtpHost)
	taskEmail := fmt.Sprintf("task+%d@%s.planfix.ru", planfixTaskId, c.Config.PlanfixAccount)
	testEmail := c.Config.SmtpEmailFrom
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
		c.Config.SmtpEmailFrom,
		taskEmail,
		c.Config.PlanfixAnaliticTypeValue,
		mins,
		c.Config.PlanfixAuthorName,
		date,
	)
	msg := []byte(body)
	return smtp.SendMail(fmt.Sprintf("%s:%d", c.Config.SmtpHost, c.Config.SmtpPort), auth, c.Config.SmtpEmailFrom, to, msg)
}

func (c TogglClient) getAnaliticData(name, typeName, countName, commentName, dateName, usersName string) (PlanfixAnaliticData, error) {
	if analiticDataCached.Id != 0 { // only on first call
		return analiticDataCached, nil
	}

	analitic, err := c.PlanfixApi.GetAnaliticByName(name)
	if err != nil {
		return PlanfixAnaliticData{}, err
	}
	analiticOptions, err := c.PlanfixApi.AnaliticGetOptions(analitic.Id)
	if err != nil {
		return PlanfixAnaliticData{}, err
	}

	analiticData := PlanfixAnaliticData{
		Id: analitic.Id,
	}

	for _, field := range analiticOptions.Analitic.Fields {
		if field.Name == typeName {
			analiticData.TypeId = field.Id
			record, err := c.PlanfixApi.GetHandbookRecordByName(field.HandbookId, c.Config.PlanfixAnaliticTypeValue)
			if err != nil {
				return analiticData, err
			}
			analiticData.TypeValueId = record.Key
		}
		if field.Name == countName {
			analiticData.CountId = field.Id
		}
		if field.Name == commentName {
			analiticData.CommentId = field.Id
		}
		if field.Name == dateName {
			analiticData.DateId = field.Id
		}
		if field.Name == usersName {
			analiticData.UsersId = field.Id
		}
	}

	analiticDataCached = analiticData
	return analiticData, nil
}

func (c TogglClient) sendWithPlanfixApi(planfixTaskId int, date string, mins int, comment string) error {
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
	userIds := struct {
		Id []int `xml:"id"`
	}{[]int{c.Config.PlanfixUserId}}

	_, err = c.PlanfixApi.ActionAdd(planfix.XmlRequestActionAdd{
		TaskGeneral: planfixTaskId,
		Description: "",
		Analitics: []planfix.XmlRequestAnalitic{
			{
				Id: analiticData.Id,
				// аналитика должна содержать поля: вид работы, кол-во, дата, коммент, юзеры
				ItemData: []planfix.XmlRequestAnaliticField{
					{FieldId: analiticData.TypeId, Value: analiticData.TypeValueId}, // name
					{FieldId: analiticData.CountId, Value: mins},                    // count, минут
					{FieldId: analiticData.CommentId, Value: comment},               // comment
					{FieldId: analiticData.DateId, Value: date},                     // date
					{FieldId: analiticData.UsersId, Value: userIds},                 // user
				},
			},
		},
	})
	return err
}
