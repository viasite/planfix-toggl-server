package client

import (
	"github.com/jason0x43/go-toggl"
	"fmt"
	"math"
	"log"
	"net/smtp"
	"github.com/viasite/planfix-toggl-server/app/config"
)

// данные не меняются при этой опции
var testMode = false

type TogglClient struct {
	Session    toggl.Session
	Config     config.Config
}

// получает записи из Toggl и отправляет в Планфикс
func (c TogglClient) SendToPlanfix() (entries){
	fmt.Println("not implemented yet")

	pendingEntries := c.GetPendingEntries()
	entries := c.GroupEntriesByTask(pendingEntries)
	for _, entry := range entries {
		entryString := entry.description + " (" + math.Floor(entry.dur / 60000  + .5) + ")"
		err := c.sendEntry(entry.planfix.task_id, entry)
		if(err != nil) {
			log.Println("[WARN] entry " + entryString + " failed to send")
		} else {
			log.Println("entry " + entryString + " sent to #" + entry.planfix.task_id)
		}
	}
	return entries;
}

func (c TogglClient) GroupEntriesByTask(entries) (grouped){
	fmt.Println("not implemented yet")
	grouped := {}
	for _, entry := range entries {
		if(grouped.hasOwnProperty(entry.planfix.task_id)){
			grouped[entry.planfix.task_id].dur += entry.dur
			grouped[entry.planfix.task_id].planfix.group_count += 1;
		} else {
			grouped[entry.planfix.task_id] = entry
		}
	}
	return Object.values(grouped)
}

func (c TogglClient) GetUserData() (account){
	account, err := c.Session.GetAccount()
	if err != nil {
		println("error:", err)
	}
	return account
}

// native toggl report
func (c TogglClient) GetReport(opts) (report, err){
	fmt.Println("not implemented yet")
	if(!opts.workspace_id){
		opts.workspace_id = c.Config.WorkspaceId
	}
	report, err := c.Session.GetDetailedReport(opts)
	return report, err
}

// report entries with planfix data
func (c TogglClient) GetEntries(opts) (entries, error) {
	report := c.getReport(opts);
	entries := []

	for _, entry in range report.data{
		entry.planfix = {
			sent: false,
			task_id: 0,
			group_count: 1
		}

		for _, tag in range entry.tags{
			// only digit == planfix.task_id
			if (tag.match(/^\d+$/)) {
				entry.planfix.task_id = parseInt(tag)
			}

			// sent tag
			if (tag == c.Config.SentTag) {
				entry.planfix.sent = true
			}
		}
	}
}

func filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func (c TogglClient) GetPendingEntries() (entries){
	user := c.GetUserData()
	entries := c.GetEntries({})
	entries = filter(entries, func(entry){ return entry.planfix.task_id != 0 })
	entries = filter(entries, func(entry){ return !entry.planfix.sent })
	entries = filter(entries, func(entry){ return entry.uid == user.id })
	return entries
}

// отправка письма и пометка тегом sent в Toggl
func (c TogglClient) sendEntry(planfixTaskId, entry) (err){
	mins := math.Floor(entry.dur / 60000  + .5);
	if(testMode){
		return nil
	}

	auth := smtp.PlainAuth("", c.Config.SmtpLogin, c.Config.SmtpPassword, c.Config.SmtpHost)
	taskEmail := "task + " + planfixTaskId + "@" + c.Config.PlanfixAccount + ".planfix.ru"
	to := []string{taskEmail}
	msg := []byte("To: " + taskEmail + "\r\n" +
		"Subject: @toggl @nonotify\r\n" +
		"\r\n" +
		"Вид работы: \r\n" + c.Config.PlanfixAnaliticName + "\r\n" +
		"time: \r\n" + mins + "\r\n" +
		"Автор: \r\n" + c.Config.PlanfixAuthorName + "\r\n" +
		"Дата:" + entry.start[0:10] + "\r\n")
	err := smtp.SendMail(fmt.Sprintf("%s:%d", c.Config.SmtpHost, c.Config.SmtpPort), auth, c.Config.EmailFrom, to, msg)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("entry [" + entry.project + "] " + entry.description + " (" + mins + ") sent to Planfix")

	if _, err := c.Session.AddRemoveTag(entry.id, c.Config.SentTag, true); err != nil {
		log.Fatal(err)
		return err
	}
}
