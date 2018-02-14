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
)

// данные не меняются при этой опции
var testMode = true

type TogglClient struct {
	Session toggl.Session
	Config  config.Config
}

type PlanfixEntryData struct {
	Sent       bool `json:"sent"`
	TaskId     int  `json:"task_id"`
	GroupCount int  `json:"group_count"`
}

type TogglPlanfixEntry struct {
	ID          int              `json:"id,omitempty"`
	Pid         int              `json:"pid"`
	Uid         int              `json:"uid"`
	Description string           `json:"description,omitempty"`
	Project     string           `json:"project,omitempty"`
	Tags        []string         `json:"tags"`
	Start       *time.Time       `json:"start,omitempty"`
	Stop        *time.Time       `json:"stop,omitempty"`
	Duration    int64            `json:"dur,omitempty"`
	Planfix     PlanfixEntryData `json:"planfix"`
}

// получает записи из Toggl и отправляет в Планфикс
func (c TogglClient) SendToPlanfix() (entries []TogglPlanfixEntry, err error) {
	fmt.Println("not implemented yet")

	pendingEntries, err := c.GetPendingEntries()
	if err != nil {
		return []TogglPlanfixEntry{}, err
	}
	entries = c.GroupEntriesByTask(pendingEntries)
	for _, entry := range entries {
		entryString := fmt.Sprintf("%s (%d)", entry.Description, math.Floor(float64(entry.Duration)/60000+.5))
		err := c.sendEntry(entry.Planfix.TaskId, entry)
		if err != nil {
			log.Println("[WARN] entry %s failed to send", entryString)
		} else {
			log.Println("entry %s sent to #%d", entryString, entry.Planfix.TaskId)
		}
	}
	return entries, nil
}

func (c TogglClient) GroupEntriesByTask(entries []TogglPlanfixEntry) (grouped []TogglPlanfixEntry) {
	g := make(map[int]TogglPlanfixEntry)
	for _, entry := range entries {
		if ge, ok := g[entry.Planfix.TaskId]; ok {
			ge.Duration += entry.Duration
			ge.Planfix.GroupCount += 1
			g[entry.Planfix.TaskId] = ge
		} else {
			grouped[entry.Planfix.TaskId] = entry
		}
	}

	grouped = make([]TogglPlanfixEntry, 0, len(g))
	for _, entry := range g {
		grouped = append(grouped, entry)
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

// native toggl report
// TODO: opts
func (c TogglClient) GetDetailedReport() (toggl.DetailedReport, error) {
	report, err := c.Session.GetDetailedReport(
		c.Config.WorkspaceId,
		"2018-02-13",
		"2018-02-14",
	)
	return report, err
}

// report entries with planfix data
// TODO: opts
func (c TogglClient) GetEntries() (entries []TogglPlanfixEntry, err error) {
	report, err := c.GetDetailedReport();
	if err != nil {
		log.Fatal("[ERROR] %s", err)
	}

	for _, entry := range report.Data {

		pfe := TogglPlanfixEntry{
			ID:          entry.ID,
			Pid:         entry.Pid,
			Uid:         entry.Uid,
			Description: entry.Description,
			Project:     entry.Project,
			Tags:        entry.Tags,
			Start:       entry.Start,
			Stop:        entry.End,
			Duration:    entry.Duration,
			Planfix: PlanfixEntryData{
				Sent:       false,
				TaskId:     0,
				GroupCount: 1,
			},
		}

		for _, tag := range entry.Tags {
			// only digit == planfix.task_id
			regex := regexp.MustCompile("/^\\d+$/")
			if regex.MatchString(tag) {
				pfe.Planfix.TaskId, _ = strconv.Atoi(tag)
			}

			// sent tag
			if tag == c.Config.SentTag {
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
	fmt.Println("not implemented yet")
	user := c.GetUserData()
	entries, err := c.GetEntries()
	if err != nil {
		return []TogglPlanfixEntry{}, err
	}
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return entry.Planfix.TaskId != 0 })
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return !entry.Planfix.Sent })
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return entry.Uid == user.Data.ID })
	return entries, nil
}

// отправка письма и пометка тегом sent в Toggl
func (c TogglClient) sendEntry(planfixTaskId int, entry TogglPlanfixEntry) (error) {
	fmt.Println("not implemented yet")
	mins := math.Floor(float64(entry.Duration)/60000 + .5)
	if testMode {
		return nil
	}

	auth := smtp.PlainAuth("", c.Config.SmtpLogin, c.Config.SmtpPassword, c.Config.SmtpHost)
	taskEmail := fmt.Sprintf("task+%d@%s.planfix.ru", planfixTaskId, c.Config.PlanfixAccount)
	to := []string{taskEmail}
	msg := []byte(fmt.Sprintf(
		"To: %s\r\n"+
			"Subject: @toggl @nonotify\r\n"+
			"\r\n"+
			"Вид работы: %s\r\n"+
			"time: %d\r\n"+
			"Автор: %s\r\n"+
			"Дата: %s\r\n",
		taskEmail,
		c.Config.PlanfixAnaliticName,
		mins,
		c.Config.PlanfixAuthorName,
		entry.Start.Format("2006-01-02"),
	))
	err := smtp.SendMail(fmt.Sprintf("%s:%d", c.Config.SmtpHost, c.Config.SmtpPort), auth, c.Config.EmailFrom, to, msg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("entry [%s] %s (%d) sent to Planfix", entry.Project, entry.Description, mins)

	if _, err := c.Session.AddRemoveTag(entry.ID, c.Config.SentTag, true); err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}
