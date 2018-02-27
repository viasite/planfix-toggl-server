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

type TogglClient struct {
	Session    toggl.Session
	Config     config.Config
	PlanfixApi planfix.Api
}

type PlanfixEntryData struct {
	Sent       bool `json:"sent"`
	TaskId     int  `json:"task_id"`
	GroupCount int  `json:"group_count"`
}

type TogglPlanfixEntry struct {
	ID              int              `json:"id,omitempty"`
	Pid             int              `json:"pid"`
	Uid             int              `json:"uid"`
	Description     string           `json:"description,omitempty"`
	Project         string           `json:"project"`
	ProjectColor    string           `json:"project_color"`
	ProjectHexColor string           `json:"project_hex_color"`
	Client          string           `json:"client,omitempty"`
	Tags            []string         `json:"tags"`
	Start           *time.Time       `json:"start,omitempty"`
	Stop            *time.Time       `json:"stop,omitempty"`
	Duration        int64            `json:"dur,omitempty"`
	Planfix         PlanfixEntryData `json:"planfix"`
}

type TogglPlanfixEntryGroup struct {
	Entries []TogglPlanfixEntry
	Description string
	Project string
	ProjectHexColor string
	Duration int64
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

// получает записи из Toggl и отправляет в Планфикс
// * нужна, чтобы сохранился c.PlanfixApi.Sid при авторизации
func (c *TogglClient) SendToPlanfix() (sumEntries []TogglPlanfixEntry, err error) {
	log.Println("[INFO] send to planfix")
	pendingEntries, err := c.GetPendingEntries()
	if err != nil {
		return []TogglPlanfixEntry{}, err
	}
	grouped := c.GroupEntriesByTask(pendingEntries)
	for taskId, entries := range grouped {
		err := c.sendEntries(taskId, entries)
		if err != nil {
			log.Printf("[WARN] entries of task #%d failed to send", taskId)
		} else {
			log.Printf("[INFO] entries sent to https://%s.planfix.ru/task/%d", c.Config.PlanfixAccount, taskId)
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
		log.Printf("[ERROR] Toggl: %s", err)
	}

	for _, entry := range report.Data {

		pfe := TogglPlanfixEntry{
			ID:              entry.ID,
			Pid:             entry.Pid,
			Uid:             entry.Uid,
			Description:     entry.Description,
			Project:         entry.Project,
			ProjectColor:    entry.ProjectColor,
			ProjectHexColor: entry.ProjectHexColor,
			Client:          entry.Client,
			Tags:            entry.Tags,
			Start:           entry.Start,
			Stop:            entry.End,
			Duration:        entry.Duration,
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
	log.Printf("[INFO] sending %s", entryString)

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
		log.Printf("[ERROR] %v", err)
		return err
	}

	// mark as sent in toggl
	for _, entry := range entries {
		entryString := fmt.Sprintf(
			"[%s] %s (%d)",
			entry.Project,
			entry.Description,
			int(math.Floor(float64(entry.Duration)/60000 + .5)),
		)
		log.Printf("[DEBUG] marking %s in toggl", entryString)
		if _, err := c.Session.AddRemoveTag(entry.ID, c.Config.TogglSentTag, true); err != nil {
			log.Fatal(err)
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
		c.Config.PlanfixAnaliticName,
		mins,
		c.Config.PlanfixAuthorName,
		date,
	)
	msg := []byte(body)
	return smtp.SendMail(fmt.Sprintf("%s:%d", c.Config.SmtpHost, c.Config.SmtpPort), auth, c.Config.SmtpEmailFrom, to, msg)
}

func (c TogglClient) sendWithPlanfixApi(planfixTaskId int, date string, mins int, comment string) error {
	analiticId := 263 // выработка
	nameId := "725"   // поминутное программирование
	userIds := struct {
		Id []int `xml:"id"`
	}{[]int{c.Config.PlanfixUserId}}

	_, err := c.PlanfixApi.ActionAdd(planfix.XmlRequestActionAdd{
		TaskGeneral: planfixTaskId,
		Description: "",
		Analitics: []planfix.XmlRequestAnalitic{
			{
				Id: analiticId,
				// аналитика должна содержать поля: вид работы, кол-во, дата, коммент, юзеры
				ItemData: []planfix.XmlRequestAnaliticField{
					{FieldId: 741, Value: nameId},  // name
					{FieldId: 747, Value: mins},    // count, минут
					{FieldId: 749, Value: comment}, // comment
					{FieldId: 743, Value: date},    // date
					{FieldId: 846, Value: userIds}, // user
				},
			},
		},
	})
	return err
}
