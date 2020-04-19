package client

import (
	"fmt"
	"github.com/gen2brain/beeep"
	"github.com/popstas/go-toggl"
	"github.com/popstas/planfix-go/planfix"
	"github.com/viasite/planfix-toggl-server/app/config"
	"io/ioutil"
	"log"
	"math"
	"net/smtp"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TogglSession interface {
	GetAccount() (toggl.Account, error)
	AddRemoveTag(entryID int, tag string, add bool) (toggl.TimeEntry, error)
	GetCurrentTimeEntry() (toggl.TimeEntry, error)
	GetSummaryReport(workspace int, since, until string) (toggl.SummaryReport, error)
	GetDetailedReport(workspace int, since, until string, page int) (toggl.DetailedReport, error)
	GetDetailedReportV2(rp toggl.DetailedReportParams) (toggl.DetailedReport, error)
	GetTagByName(name string, wid int) (tag toggl.Tag, err error)
	GetWorkspaces() (workspaces []toggl.Workspace, err error)
}

// TogglClient - Клиент, общающийся с Toggl и Планфиксом
type TogglClient struct {
	Session      TogglSession
	Config       *config.Config
	PlanfixAPI   planfix.API
	Logger       *log.Logger
	analiticData PlanfixAnaliticData
	sentLog		 map[string]int
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

// Run запускает фоновые процессы
func (c TogglClient) Run() {
	// init map
	c.sentLog = make(map[string]int)
	// start tag cleaner
	go c.RunTagCleaner()
	// start sender
	go c.RunSender()
}

// RunSender - запускалка цикла отправки накопившихся toggl-записей
func (c TogglClient) RunSender() {
	if c.Config.SendInterval <= 0 {
		c.Logger.Println("[INFO] Интервал отправки не установлен, периодическая отправка отключена")
		return
	}

	time.Sleep(1 * time.Second) // wait for server start
	for {
		c.SendToPlanfix()
		time.Sleep(time.Duration(c.Config.SendInterval) * time.Minute)
	}
}

// ReloadConfig - пересоздает API из последней версии конфига
func (c *TogglClient) ReloadConfig() {
	c.PlanfixAPI = planfix.New(
		c.Config.PlanfixAPIUrl,
		c.Config.PlanfixAPIKey,
		c.Config.PlanfixAccount,
		c.Config.PlanfixUserName,
		c.Config.PlanfixUserPassword,
	)
	if !c.Config.Debug {
		c.PlanfixAPI.Logger.SetFlags(0)
		c.PlanfixAPI.Logger.SetOutput(ioutil.Discard)
	}
	c.PlanfixAPI.UserAgent = "planfix-toggl"

	sess := toggl.OpenSession(c.Config.TogglAPIToken)
	c.Session = &sess
}

// RunTagCleaner - запускалка цикла очистки запущенных toggl-записей от тега sent
func (c TogglClient) RunTagCleaner() {
	time.Sleep(1 * time.Second) // wait for server start
	for {
		entry, err := c.Session.GetCurrentTimeEntry()
		if err != nil {
			c.Logger.Println("[ERROR] не удалось получить текущую toggl-запись")
			continue
		}

		// delete sent tag
		for _, tag := range entry.Tags {
			if tag == c.Config.TogglSentTag {
				c.Logger.Printf("[INFO] убран тег %s из текущей записи %s", c.Config.TogglSentTag, entry.Description)
				c.Session.AddRemoveTag(entry.ID, c.Config.TogglSentTag, false)
			}
		}

		time.Sleep(1 * time.Minute)
	}
}

// SendToPlanfix получает записи из Toggl и отправляет в Планфикс
// * нужна, чтобы сохранился c.PlanfixAPI.Sid при авторизации
func (c *TogglClient) SendToPlanfix() (err error) {
	c.Logger.Println("[INFO] отправка в Планфикс")
	pendingEntries, err := c.GetPendingEntries()
	if err != nil {
		return err
	}
	c.Logger.Printf("[INFO] в очереди на отправку: %d", len(pendingEntries))
	days := c.GroupEntriesByDay(pendingEntries)
	for day, dayEntries := range days {
		tasks := c.GroupEntriesByTask(dayEntries)
		minsTotal := 0

		for taskID, entries := range tasks {
			err, mins := c.sendEntries(taskID, entries)

			minsTotal += mins
			// add to day time
			if dayMins, ok := c.sentLog[day]; ok {
				c.sentLog[day] = dayMins + mins
			} else {
				c.sentLog[day] = mins
			}

			taskURL := fmt.Sprintf("https://%s.planfix.ru/task/%d", c.Config.PlanfixAccount, taskID)
			if err != nil {
				c.Logger.Printf("[ERROR] записи к задаче %s (%s) не удалось отправить: %s", taskURL, day, err)
			} else {
				c.Logger.Printf("[INFO] %d минут отправлены на %s (%s)", mins, taskURL, day)
			}
		}
		dayHours := float32(c.sentLog[day]) / 60
		c.Logger.Printf("[INFO] минут: %d, задач: %d, всего %.1f часов за %s", minsTotal, len(tasks), dayHours, day)
		c.Notify(fmt.Sprintf("Sent %d minutes to %d tasks\n%.1f hours for %s", minsTotal, len(tasks), dayHours, day))
	}
	return nil
}

// GroupEntriesByDay объединяет плоский список toggl-записей в map c ключом - Y-m-d
func (c TogglClient) GroupEntriesByDay(entries []TogglPlanfixEntry) (grouped map[string][]TogglPlanfixEntry) {
	grouped = make(map[string][]TogglPlanfixEntry)
	if len(entries) == 0 {
		return grouped
	}
	for _, entry := range entries {
		day := entry.Start.Format("02-01-2006")
		grouped[day] = append(grouped[day], entry)
	}
	return grouped
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

// SumEntriesGroup объединяет несколько toggl-записей в одну с просуммированным временем
// Входной map формируется через GroupEntriesByTask
// Ключ массива - ID задачи в Планфиксе
func (c TogglClient) SumEntriesGroup(grouped map[int][]TogglPlanfixEntry) (summed []TogglPlanfixEntry) {
	g := make(map[int]TogglPlanfixEntry)
	for taskID, entries := range grouped {
		for _, entry := range entries {
			if ge, ok := g[taskID]; ok {
				ge.Duration += entry.Duration
				ge.Planfix.GroupCount++
				g[entry.Planfix.TaskID] = ge
			} else {
				entry.Description = c.getSumEntryName(entries)
				g[entry.Planfix.TaskID] = entry
			}
		}
	}

	taskIDs := make([]int, 0, len(g))
	for k := range g {
		taskIDs = append(taskIDs, k)
	}
	sort.Ints(taskIDs)

	summed = make([]TogglPlanfixEntry, 0, len(g))
	for _, taskID := range taskIDs {
		summed = append(summed, g[taskID])
	}

	return summed
}

func (c TogglClient) getSumEntryName(entries []TogglPlanfixEntry) string {
	names := []string{}
	for _, entry := range entries {
		names = append(names, entry.Description)
	}
	// sort
	sort.Strings(names)

	// group
	groupNamesCounts := make(map[string]int)
	for _, name := range names {
		groupNamesCounts[name]++
	}

	// keys
	names = []string{}
	for name, _ := range groupNamesCounts {
		names = append(names, name)
	}

	return strings.Join(names, "\n")
}

// GetTogglUser возвращает юзера в Toggl
func (c TogglClient) GetTogglUser() (account toggl.Account, err error) {
	account, err = c.Session.GetAccount()
	if err != nil {
		return account, fmt.Errorf("Не удалось получить Toggl UserID, проверьте TogglAPIToken, %s", err.Error())
	}
	return account, nil
}

// IsWorkspaceExists проверяет наличие workspace в доступных
func (c TogglClient) IsWorkspaceExists(wid int) (bool, error) {
	ws, err := c.Session.GetWorkspaces()
	if err != nil {
		return false, fmt.Errorf("Не удалось получить Toggl workspaces, проверьте TogglAPIToken, %s", err.Error())
	}
	for _, w := range ws {
		if w.ID == wid {
			return true, nil
		}
	}
	return false, nil
}

// GetPlanfixUser возвращает юзера в Планфиксе
func (c TogglClient) GetPlanfixUser() (user planfix.XMLResponseUser, err error) {
	var userResponse planfix.XMLResponseUserGet
	userResponse, err = c.PlanfixAPI.UserGet(0)
	user = userResponse.User
	if err != nil {
		return user, fmt.Errorf("Не удалось получить Planfix UserID, проверьте PlanfixAPIKey, PlanfixAPIUrl, PlanfixUserName, PlanfixUserPassword, %s", err.Error())
	}
	return user, nil
}

func (c TogglClient) ReportToTogglPlanfixEntry(report toggl.DetailedReport) (entries []TogglPlanfixEntry) {
	for _, entry := range report.Data {
		pfe := c.togglDetailedEntryToPlanfixTogglEntry(entry)
		entries = append(entries, pfe)
	}
	return entries
}

func (c TogglClient) togglEntryToTogglDetailedEntry(entry toggl.TimeEntry) toggl.DetailedTimeEntry{
	return toggl.DetailedTimeEntry{
		ID: entry.ID,
		Pid: entry.Pid,
		Tid: entry.Tid,
		Description: entry.Description,
		Start: entry.Start,
		End: entry.Stop,
		Tags: entry.Tags,
		Duration: entry.Duration,
		Billable: entry.Billable,
	}
}

func (c TogglClient) togglDetailedEntryToPlanfixTogglEntry(entry toggl.DetailedTimeEntry) (pfe TogglPlanfixEntry){
	pfe = TogglPlanfixEntry{
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

	return pfe
}

// GetEntries получает []toggl.DetailedTimeEntry и превращает их в []TogglPlanfixEntry с подмешенными данными Планфикса
func (c TogglClient) GetCurrentEntry() (entry TogglPlanfixEntry, err error) {
	togglEntry, err := c.Session.GetCurrentTimeEntry()
	detailedEntry := c.togglEntryToTogglDetailedEntry(togglEntry)
	entry = c.togglDetailedEntryToPlanfixTogglEntry(detailedEntry)
	return entry, err
}

// GetEntries получает []toggl.DetailedTimeEntry и превращает их в []TogglPlanfixEntry с подмешенными данными Планфикса
func (c TogglClient) GetEntries(togglWorkspaceID int, since, until string) (entries []TogglPlanfixEntry, err error) {
	report, err := c.Session.GetDetailedReport(togglWorkspaceID, since, until, 1)
	if err != nil {
		c.Logger.Printf("[ERROR] Toggl: %s", err)
	}

	entries = c.ReportToTogglPlanfixEntry(report)
	return entries, nil
}

func (c TogglClient) GetReport(rp toggl.DetailedReportParams) toggl.DetailedReport {
	rp.WorkspaceID = c.Config.TogglWorkspaceID
	rp.Rounding = true
	report, err := c.Session.GetDetailedReportV2(rp)
	if err != nil {
		c.Logger.Printf("[ERROR] Toggl: %s", err)
	}
	return report
}

func (c TogglClient) GetReportV1(togglWorkspaceID int, since, until string, page int) toggl.DetailedReport {
	report, err := c.Session.GetDetailedReport(togglWorkspaceID, since, until, page)
	if err != nil {
		c.Logger.Printf("[ERROR] Toggl: %s", err)
	}
	return report
}

func (c TogglClient) getDetailedReportParams() {
	return
}

func (c TogglClient) GetEntriesV2(rp toggl.DetailedReportParams) (entries []TogglPlanfixEntry, err error) {
	report := c.GetReport(rp)
	entries = c.ReportToTogglPlanfixEntry(report)
	return entries, nil
}

func (c TogglClient) GetEntriesByTag(tagName string) (entries []TogglPlanfixEntry, err error) {
	tag, err := c.Session.GetTagByName(tagName, c.Config.TogglWorkspaceID)
	if err != nil {
		return entries, err
	}
	report := c.GetReport(toggl.DetailedReportParams{
		// TODO: defaultReportParams
		TagIDs: []int{tag.ID},
	})
	entries = c.ReportToTogglPlanfixEntry(report)
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

// GetPendingEntriesPage возвращает toggl-записи, которые должны быть отправлены в Планфикс для конкретной страницы
func (c TogglClient) getPendingEntriesPage(page int) (entries []TogglPlanfixEntry, err error) {
	entries, err = c.GetEntriesV2(toggl.DetailedReportParams{
		Page: page,
	})
	if err != nil {
		return []TogglPlanfixEntry{}, err
	}
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return entry.Planfix.TaskID != 0 })
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return !entry.Planfix.Sent })
	entries = filter(entries, func(entry TogglPlanfixEntry) bool { return entry.Uid == c.Config.TogglUserID })
	return entries, nil
}

// GetPendingEntries возвращает toggl-записи, которые должны быть отправлены в Планфикс
func (c TogglClient) GetPendingEntries() (entries []TogglPlanfixEntry, err error) {
	maxPages := 20
	for currentPage := 1; currentPage <= maxPages; currentPage++ {
		pageEntries, err := c.getPendingEntriesPage(currentPage)
		if err != nil {
			return entries, err
		}
		if len(pageEntries) == 0 {
			break;
		}
		entries = append(entries, pageEntries...)
	}
	return entries, err
}

// sendEntries отправляет toggl-записи в Планфикс и помечает их в Toggl тегом sent
func (c TogglClient) sendEntries(planfixTaskID int, entries []TogglPlanfixEntry) (error, int) {
	// будет точно просуммировано в одну
	sumEntry := c.SumEntriesGroup(map[int][]TogglPlanfixEntry{
		planfixTaskID: entries,
	})[0]

	date := sumEntry.Start.Format("2006-01-02")
	mins := int(math.Floor(float64(sumEntry.Duration)/60000 + .5))
	entryString := fmt.Sprintf(
		"[%s] %s (%d)",
		sumEntry.Project,
		sumEntry.Description,
		mins,
	)
	comment := fmt.Sprintf("toggl: %s", entryString)

	if c.Config.Debug {
		c.Logger.Printf("[DEBUG] sending %s", entryString)
	}
	if c.Config.DryRun {
		c.Logger.Println("[DEBUG] dry-run")
		return nil, mins
	}

	// send to planfix
	var err error
	if c.Config.PlanfixUserID != 0 {
		err = c.sendWithPlanfixAPI(planfixTaskID, date, mins, comment)
	} else {
		err = c.sendWithSMTP(planfixTaskID, date, mins)
	}
	if err != nil {
		c.Logger.Printf("[ERROR] %v", err)
		return err, mins
	}

	return c.markAsSent(entries), mins
}

// markAsSent отмечает toggl-записи тегом sent
func (c TogglClient) markAsSent(entries []TogglPlanfixEntry) error {
	for _, entry := range entries {
		entryString := fmt.Sprintf(
			"[%s] %s (%d)",
			entry.Project,
			entry.Description,
			int(math.Floor(float64(entry.Duration)/60000+.5)),
		)
		if c.Config.Debug {
			c.Logger.Printf("[DEBUG] marking %s in toggl", entryString)
		}
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
	taskEmail := c.getTaskEmail(planfixTaskID)
	to := []string{taskEmail}
	if c.Config.Debug {
		testEmail := c.Config.SMTPEmailFrom
		to = append(to, testEmail)
	}
	body := c.getEmailBody(planfixTaskID, date, mins)
	msg := []byte(body)
	return smtp.SendMail(fmt.Sprintf("%s:%d", c.Config.SMTPHost, c.Config.SMTPPort), auth, c.Config.SMTPEmailFrom, to, msg)
}

// getTaskEmail возвращает email задачи по ее номеру
func (c TogglClient) getTaskEmail(planfixTaskID int) string {
	return fmt.Sprintf("task+%d@%s.planfix.ru", planfixTaskID, c.Config.PlanfixAccount)
}

// getEmailBody возвращает email body для отправки в Планфикс
func (c TogglClient) getEmailBody(planfixTaskID int, date string, mins int) string {
	taskEmail := c.getTaskEmail(planfixTaskID)
	return fmt.Sprintf(
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
}

// sendWithPlanfixAPI отправляет toggl-запись через Планфикс API
func (c TogglClient) sendWithPlanfixAPI(planfixTaskID int, date string, mins int, comment string) error {
	userIDs := struct {
		ID []int `xml:"id"`
	}{[]int{c.Config.PlanfixUserID}}

	_, err := c.PlanfixAPI.ActionAdd(planfix.XMLRequestActionAdd{
		TaskGeneral: planfixTaskID,
		Description: "",
		Analitics: []planfix.XMLRequestActionAnalitic{
			{
				ID: c.analiticData.ID,
				// аналитика должна содержать поля: вид работы, кол-во, дата, коммент, юзеры
				ItemData: []planfix.XMLRequestAnaliticField{
					{FieldID: c.analiticData.TypeID, Value: c.analiticData.TypeValueID}, // name
					{FieldID: c.analiticData.CountID, Value: mins},                      // count, минут
					{FieldID: c.analiticData.CommentID, Value: comment},                 // comment
					{FieldID: c.analiticData.DateID, Value: date},                       // date
					{FieldID: c.analiticData.UsersID, Value: userIDs},                   // user
				},
			},
		},
	})
	return err
}

// GetAnaliticDataCached получает аналитику из кеша (по возможности)
func (c *TogglClient) GetAnaliticDataCached(name, typeName, typeValue, countName, commentName, dateName, usersName string) (PlanfixAnaliticData, error) {
	if c.analiticData.ID != 0 { // only on first call
		return c.analiticData, nil
	}
	data, err := c.GetAnaliticData(name, typeName, typeValue, countName, commentName, dateName, usersName)
	c.analiticData = data
	return c.analiticData, err
}

// GetAnaliticData получает ID аналитики и ее полей из названий аналитики и полей
func (c *TogglClient) GetAnaliticData(name, typeName, typeValue, countName, commentName, dateName, usersName string) (PlanfixAnaliticData, error) {
	// получение аналитики
	analitic, err := c.PlanfixAPI.GetAnaliticByName(name)
	if err != nil {
		return PlanfixAnaliticData{}, err
	}

	// получение полей аналитики
	analiticOptions, err := c.PlanfixAPI.AnaliticGetOptions(analitic.ID)
	if err != nil {
		return PlanfixAnaliticData{}, err
	}

	analiticData := PlanfixAnaliticData{
		ID: analitic.ID,
	}

	// получение ID полей по их названиям
	for _, field := range analiticOptions.Analitic.Fields {
		switch field.Name {
		case typeName:
			analiticData.TypeID = field.ID
			// получение ID записи справочника
			record, _ := c.PlanfixAPI.GetHandbookRecordByName(field.HandbookID, typeValue)
			analiticData.TypeValueID = record.Key
		case countName:
			analiticData.CountID = field.ID
		case commentName:
			analiticData.CommentID = field.ID
		case dateName:
			analiticData.DateID = field.ID
		case usersName:
			analiticData.UsersID = field.ID
		}
	}

	err = c.isAnaliticValid(analiticData)
	return analiticData, err
}

// isAnaliticValid проходит по всем полям структуры PlanfixAnaliticData и возвращает ошибку, если хоть один ID == 0
func (c TogglClient) isAnaliticValid(data PlanfixAnaliticData) error {
	var errors []string
	v := reflect.ValueOf(data)
	typeOf := v.Type()
	//values := make([]interface{}, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		name := typeOf.Field(i).Name
		value := v.Field(i).Int()
		if value == 0 {
			errors = append(errors, fmt.Sprintf("%s not found", name))
		}
	}
	if len(errors) != 0 {
		return fmt.Errorf(strings.Join(errors, ", "))
	}
	return nil
}

func (c TogglClient) Notify(msg string) error {
	err := beeep.Notify("", msg, "assets/icon.png")
	return err
}