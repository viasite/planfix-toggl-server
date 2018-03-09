package client

import (
	"bytes"
	"encoding/xml"
	"github.com/popstas/go-toggl"
	"github.com/popstas/planfix-go/planfix"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/viasite/planfix-toggl-server/app/config"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var output bytes.Buffer

type planfixRequestStruct struct {
	XMLName xml.Name `xml:"request"`
	Method  string   `xml:"method,attr"`
	Account string   `xml:"account"`
	Sid     string   `xml:"sid"`
}

func fixtureFromFile(fixtureName string) string {
	buf, _ := ioutil.ReadFile("../../tests/fixtures/" + fixtureName)
	return string(buf)
}

type MockedServer struct {
	*httptest.Server
	Requests  [][]byte
	Responses []string // fifo queue of answers
}

func NewMockedServer(responses []string) *MockedServer {
	s := &MockedServer{
		Requests:  [][]byte{},
		Responses: responses,
	}

	s.Server = httptest.NewServer(s)
	return s
}

func (s *MockedServer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	lastRequest, err := ioutil.ReadAll(req.Body)
	if err != nil {
		panic(err)
	}

	rs := planfixRequestStruct{}
	err = xml.Unmarshal(lastRequest, &rs)
	if err != nil {
		panic(err)
	}
	s.Requests = append(s.Requests, lastRequest)
	answer := s.Responses[0]

	s.Responses = s.Responses[1:]
	resp.Write([]byte(answer))
}

type MockedTogglSession struct {
	mock.Mock
	TogglSession
}

func (s *MockedTogglSession) GetAccount() (toggl.Account, error) {
	args := s.Called()
	return args.Get(0).(toggl.Account), args.Error(1)
}

func (s *MockedTogglSession) GetDetailedReport(workspace int, since, until string, page int) (toggl.DetailedReport, error) {
	args := s.Called(workspace, since, until, page)
	return args.Get(0).(toggl.DetailedReport), args.Error(1)
}

func (s *MockedTogglSession) AddRemoveTag(entryID int, tag string, add bool) (toggl.TimeEntry, error) {
	args := s.Called(entryID, tag, add)
	return args.Get(0).(toggl.TimeEntry), args.Error(1)
}

func newClient() TogglClient {
	cfg := config.Config{
		TogglSentTag: "sent",
	}
	api := planfix.New("", "apiKey", "account", "user", "password")
	api.Sid = "123"

	sess := &MockedTogglSession{}
	return TogglClient{
		Session:    sess,
		Config:     &cfg,
		PlanfixAPI: api,
		Logger:     log.New(&output, "", log.LstdFlags),
	}
}

func getTestEntries() []TogglPlanfixEntry {
	return []TogglPlanfixEntry{
		{
			toggl.DetailedTimeEntry{Duration: 1},
			PlanfixEntryData{TaskID: 1, GroupCount: 1},
		},
		{
			toggl.DetailedTimeEntry{Duration: 2},
			PlanfixEntryData{TaskID: 1, GroupCount: 1},
		},
		{
			toggl.DetailedTimeEntry{Duration: 3},
			PlanfixEntryData{TaskID: 2, GroupCount: 1},
		},
		{
			toggl.DetailedTimeEntry{Duration: 4},
			PlanfixEntryData{TaskID: 2, GroupCount: 1},
		},
		{
			toggl.DetailedTimeEntry{Duration: 5},
			PlanfixEntryData{TaskID: 2, GroupCount: 1},
		},
		{
			toggl.DetailedTimeEntry{Duration: 6},
			PlanfixEntryData{TaskID: 3, GroupCount: 1},
		},
	}
}

func getTestGroupedEntries() map[int][]TogglPlanfixEntry {
	return map[int][]TogglPlanfixEntry{
		1: {
			{
				toggl.DetailedTimeEntry{Duration: 1},
				PlanfixEntryData{TaskID: 1, GroupCount: 1},
			},
			{
				toggl.DetailedTimeEntry{Duration: 2},
				PlanfixEntryData{TaskID: 1, GroupCount: 1},
			},
		},
		2: {
			{
				toggl.DetailedTimeEntry{Duration: 3},
				PlanfixEntryData{TaskID: 2, GroupCount: 1},
			},
			{
				toggl.DetailedTimeEntry{Duration: 4},
				PlanfixEntryData{TaskID: 2, GroupCount: 1},
			},
			{
				toggl.DetailedTimeEntry{Duration: 5},
				PlanfixEntryData{TaskID: 2, GroupCount: 1},
			},
		},
		3: {
			{
				toggl.DetailedTimeEntry{Duration: 6},
				PlanfixEntryData{TaskID: 3, GroupCount: 1},
			},
		},
	}
}

func getTestDate() time.Time {
	return time.Date(2018, 3, 4, 1, 2, 3, 0, time.Local)
}

func getTestDetailedReport() toggl.DetailedReport {
	date := getTestDate()
	return toggl.DetailedReport{Data: []toggl.DetailedTimeEntry{
		// will be filtered by sent tag
		{
			ID:          1,
			Project:     "project1",
			Description: "description1",
			Tags:        []string{"12345", "sent"},
			Start:       &date,
		},
		{
			ID:          2,
			Project:     "project1",
			Description: "description1",
			Tags:        []string{"12345"},
			Start:       &date,
		},
		// will be filtered by taskID tag
		{
			ID:          3,
			Project:     "project1",
			Description: "description1",
			Tags:        []string{},
			Start:       &date,
		},
		{
			ID:          4,
			Project:     "project1",
			Description: "description1",
			Tags:        []string{"12345"},
			Start:       &date,
		},
	}}
}

func TestTogglClient_SumEntriesGroup(t *testing.T) {
	c := newClient()
	groupedEntries := getTestGroupedEntries()
	expected := []TogglPlanfixEntry{
		{
			toggl.DetailedTimeEntry{Duration: 3},
			PlanfixEntryData{TaskID: 1, GroupCount: 2},
		},
		{
			toggl.DetailedTimeEntry{Duration: 12},
			PlanfixEntryData{TaskID: 2, GroupCount: 3},
		},
		{
			toggl.DetailedTimeEntry{Duration: 6},
			PlanfixEntryData{TaskID: 3, GroupCount: 1},
		},
	}

	summed := c.SumEntriesGroup(groupedEntries)
	assert.Equal(t, expected, summed)
}

func TestTogglClient_GroupEntriesByTask(t *testing.T) {
	c := newClient()
	entries := getTestEntries()
	expected := getTestGroupedEntries()

	grouped := c.GroupEntriesByTask(entries)
	assert.Equal(t, expected, grouped)
}

func TestTogglClient_GroupEntriesByTask_empty(t *testing.T) {
	c := newClient()
	entries := []TogglPlanfixEntry{}
	expected := map[int][]TogglPlanfixEntry{}

	grouped := c.GroupEntriesByTask(entries)
	assert.Equal(t, expected, grouped)
}

func TestTogglClient_sendEntries_dryRun(t *testing.T) {
	c := newClient()
	c.Config.DryRun = true
	date := getTestDate()
	entries := []TogglPlanfixEntry{
		{
			toggl.DetailedTimeEntry{
				Duration:    60000,
				Start:       &date,
				Project:     "project",
				Description: "description",
			},
			PlanfixEntryData{TaskID: 1, GroupCount: 1},
		},
		{
			toggl.DetailedTimeEntry{Duration: 120000},
			PlanfixEntryData{TaskID: 1, GroupCount: 1},
		},
	}

	c.sendEntries(1, entries)
	assert.Contains(t, output.String(), "[DEBUG] sending [project] description (3)")
	assert.Contains(t, output.String(), "[DEBUG] dry-run")
}

func TestTogglClient_GetTogglUserID(t *testing.T) {
	c := newClient()
	sess := &MockedTogglSession{}
	c.Session = sess

	togglUser := toggl.Account{Data: struct {
		APIToken        string            `json:"api_token"`
		Timezone        string            `json:"timezone"`
		ID              int               `json:"id"`
		Workspaces      []toggl.Workspace `json:"workspaces"`
		Clients         []toggl.Client    `json:"clients"`
		Projects        []toggl.Project   `json:"projects"`
		Tasks           []toggl.Task      `json:"tasks"`
		Tags            []toggl.Tag       `json:"tags"`
		TimeEntries     []toggl.TimeEntry `json:"time_entries"`
		BeginningOfWeek int               `json:"beginning_of_week"`
	}{ID: 123}}
	sess.On("GetAccount").Return(togglUser, nil)

	togglUserID := c.GetTogglUserID()
	assert.Equal(t, 123, togglUserID)
}

func TestTogglClient_GetPlanfixUserID(t *testing.T) {
	c := newClient()
	ms := NewMockedServer([]string{
		fixtureFromFile("user.get.xml"),
		//fixtureFromFile("error.xml"),
	})
	c.PlanfixAPI.URL = ms.URL

	planfixUserID := c.GetPlanfixUserID()
	assert.Equal(t, 9230, planfixUserID)
}

func TestTogglClient_GetEntries(t *testing.T) {
	c := newClient()
	sess := &MockedTogglSession{}
	c.Session = sess

	since := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	until := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	report := toggl.DetailedReport{Data: []toggl.DetailedTimeEntry{
		{
			ID:          1,
			Project:     "project1",
			Description: "description1",
			Tags:        []string{"12345", "sent"},
		},
	}}

	sess.On("GetDetailedReport", 234, since, until, 1).Return(report, nil)

	report, _ = c.Session.GetDetailedReport(234, since, until, 1)
	entries, _ := c.GetEntries(234, since, until)

	expected := []TogglPlanfixEntry{
		{
			DetailedTimeEntry: report.Data[0],
			Planfix:           PlanfixEntryData{GroupCount: 1, Sent: true, TaskID: 12345},
		},
	}
	assert.Equal(t, expected, entries)
}

func TestTogglClient_GetPendingEntries(t *testing.T) {
	c := newClient()
	sess := &MockedTogglSession{}
	c.Session = sess

	since := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	until := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	date := getTestDate()
	report := getTestDetailedReport()

	sess.On("GetDetailedReport", c.Config.TogglWorkspaceID, since, until, 1).Return(report, nil)

	entries, _ := c.GetPendingEntries()

	expected := []TogglPlanfixEntry{
		{
			DetailedTimeEntry: toggl.DetailedTimeEntry{
				ID:          2,
				Project:     "project1",
				Description: "description1",
				Tags:        []string{"12345"},
				Start:       &date,
			},
			Planfix: PlanfixEntryData{GroupCount: 1, Sent: false, TaskID: 12345},
		},
		{
			DetailedTimeEntry: toggl.DetailedTimeEntry{
				ID:          4,
				Project:     "project1",
				Description: "description1",
				Tags:        []string{"12345"},
				Start:       &date,
			},
			Planfix: PlanfixEntryData{GroupCount: 1, Sent: false, TaskID: 12345},
		},
	}
	assert.Equal(t, expected, entries)
}

func TestTogglClient_markAsSent(t *testing.T) {
	c := newClient()
	sess := &MockedTogglSession{}
	c.Session = sess

	planfixEntries := []TogglPlanfixEntry{
		{
			DetailedTimeEntry: toggl.DetailedTimeEntry{
				ID:          2,
				Project:     "project1",
				Description: "description1",
				Tags:        []string{"12345"},
			},
			Planfix: PlanfixEntryData{GroupCount: 1, Sent: false, TaskID: 12345},
		},
		{
			DetailedTimeEntry: toggl.DetailedTimeEntry{
				ID:          4,
				Project:     "project1",
				Description: "description1",
				Tags:        []string{"12345"},
			},
			Planfix: PlanfixEntryData{GroupCount: 1, Sent: false, TaskID: 12345},
		},
	}
	togglEntries := []toggl.TimeEntry{
		{
			ID:          2,
			Description: "description1",
			Tags:        []string{"12345"},
		},
		{
			ID:          4,
			Description: "description1",
			Tags:        []string{"12345"},
		},
	}
	sess.On("AddRemoveTag", 2, c.Config.TogglSentTag, true).Return(togglEntries[0], nil)
	sess.On("AddRemoveTag", 4, c.Config.TogglSentTag, true).Return(togglEntries[1], nil)

	err := c.markAsSent(planfixEntries)
	assert.NoError(t, err)
}

func TestTogglClient_getTaskEmail(t *testing.T) {
	c := newClient()
	c.Config.PlanfixAccount = "mycompany"

	taskEmail := c.getTaskEmail(123)
	assert.Equal(t, "task+123@mycompany.planfix.ru", taskEmail)
}

func TestTogglClient_getEmailBody(t *testing.T) {
	c := newClient()
	c.Config.PlanfixAccount = "mycompany"
	c.Config.SMTPEmailFrom = "me@mycompany.ru"
	c.Config.PlanfixAnaliticTypeValue = "Название аналитики"
	c.Config.PlanfixAuthorName = "Имя Фамилия"

	expectedBody := "Content-Type: text/plain; charset=\"utf-8\"\r\n" +
		"From: me@mycompany.ru\r\n" +
		"To: task+123@mycompany.planfix.ru\r\n" +
		"Subject: @toggl @nonotify\r\n" +
		"\r\n" +
		"Вид работы: Название аналитики\r\n" +
		"time: 234\r\n" +
		"Автор: Имя Фамилия\r\n" +
		"Дата: 2018-03-04\r\n"

	body := c.getEmailBody(123, "2018-03-04", 234)
	assert.Equal(t, expectedBody, body)
}

func TestTogglClient_GetAnaliticData(t *testing.T) {
	c := newClient()
	ms := NewMockedServer([]string{
		fixtureFromFile("analitic.getList.xml"),
		fixtureFromFile("analitic.getOptions.xml"),
		fixtureFromFile("analitic.getHandbook.xml"),
	})
	c.PlanfixAPI.URL = ms.URL
	c.Config.PlanfixUserID = 123
	c.Config.PlanfixAnaliticName = "Выработка"
	c.Config.PlanfixAnaliticTypeName = "Вид работы"
	c.Config.PlanfixAnaliticTypeValue = "Поминутная работа программиста"
	c.Config.PlanfixAnaliticCountName = "Кол-во"
	c.Config.PlanfixAnaliticCommentName = "Комментарий / ссылка"
	c.Config.PlanfixAnaliticDateName = "Дата"
	c.Config.PlanfixAnaliticUsersName = "Сотрудник"

	// нормальное поведение
	analiticData, err := c.GetAnaliticData(
		c.Config.PlanfixAnaliticName,
		c.Config.PlanfixAnaliticTypeName,
		c.Config.PlanfixAnaliticTypeValue,
		c.Config.PlanfixAnaliticCountName,
		c.Config.PlanfixAnaliticCommentName,
		c.Config.PlanfixAnaliticDateName,
		c.Config.PlanfixAnaliticUsersName,
	)
	assert.NoError(t, err)
	assert.Equal(t, 725, analiticData.TypeValueID)

	// тест кеша
	analiticData, err = c.GetAnaliticData("", "", "", "", "", "", "")
	assert.NoError(t, err)
	assert.Equal(t, 725, analiticData.TypeValueID)

	// неправильный вид работы
	c.analiticData = PlanfixAnaliticData{} // сброс кеша
	ms = NewMockedServer([]string{
		fixtureFromFile("analitic.getList.xml"),
		fixtureFromFile("analitic.getOptions.xml"),
		fixtureFromFile("analitic.getHandbook.xml"),
	})
	c.PlanfixAPI.URL = ms.URL
	c.Config.PlanfixAnaliticTypeValue = "Какой-то неизвестный вид работы"
	analiticData, err = c.GetAnaliticData(
		c.Config.PlanfixAnaliticName,
		c.Config.PlanfixAnaliticTypeName,
		c.Config.PlanfixAnaliticTypeValue,
		c.Config.PlanfixAnaliticCountName,
		c.Config.PlanfixAnaliticCommentName,
		c.Config.PlanfixAnaliticDateName,
		c.Config.PlanfixAnaliticUsersName,
	)
	assert.Error(t, err)
	assert.Equal(t, 0, analiticData.TypeValueID)
	assert.Equal(t, 749, analiticData.CommentID)
}

// TODO: проходит метод полностью, но непонятно что проверяет
func TestTogglClient_sendWithPlanfixAPI(t *testing.T) {
	c := newClient()
	c.Config.PlanfixAnaliticName = "Выработка"
	c.Config.PlanfixAnaliticTypeName = "Вид работы"
	c.Config.PlanfixAnaliticTypeValue = "Поминутная работа программиста"
	c.Config.PlanfixAnaliticCountName = "Кол-во"
	c.Config.PlanfixAnaliticCommentName = "Комментарий / ссылка"
	c.Config.PlanfixAnaliticDateName = "Дата"
	c.Config.PlanfixAnaliticUsersName = "Сотрудник"

	ms := NewMockedServer([]string{
		fixtureFromFile("analitic.getList.xml"),
		fixtureFromFile("analitic.getOptions.xml"),
		fixtureFromFile("analitic.getHandbook.xml"),
		fixtureFromFile("action.add.xml"),
	})
	c.PlanfixAPI.URL = ms.URL
	c.Config.PlanfixUserID = 123
	c.Config.PlanfixAnaliticName = "Выработка"

	err := c.sendWithPlanfixAPI(123, "2018-03-04", 234, "comment")
	assert.NoError(t, err)
}

// TODO: проходит метод полностью, но непонятно что проверяет
func TestTogglClient_SendToPlanfix(t *testing.T) {
	c := newClient()
	sess := &MockedTogglSession{}
	c.Session = sess

	c.Config.PlanfixAnaliticName = "Выработка"
	c.Config.PlanfixAnaliticTypeName = "Вид работы"
	c.Config.PlanfixAnaliticTypeValue = "Поминутная работа программиста"
	c.Config.PlanfixAnaliticCountName = "Кол-во"
	c.Config.PlanfixAnaliticCommentName = "Комментарий / ссылка"
	c.Config.PlanfixAnaliticDateName = "Дата"
	c.Config.PlanfixAnaliticUsersName = "Сотрудник"

	ms := NewMockedServer([]string{
		fixtureFromFile("analitic.getList.xml"),
		fixtureFromFile("analitic.getOptions.xml"),
		fixtureFromFile("analitic.getHandbook.xml"),
		fixtureFromFile("action.add.xml"),
	})
	c.PlanfixAPI.URL = ms.URL
	c.Config.PlanfixUserID = 123
	c.Config.PlanfixAnaliticName = "Выработка"

	since := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	until := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	report := getTestDetailedReport()
	sess.On("GetDetailedReport", c.Config.TogglWorkspaceID, since, until, 1).Return(report, nil)

	sess.On("AddRemoveTag", 2, c.Config.TogglSentTag, true).Return(toggl.TimeEntry{}, nil)
	sess.On("AddRemoveTag", 4, c.Config.TogglSentTag, true).Return(toggl.TimeEntry{}, nil)

	pending, _ := c.GetPendingEntries()
	grouped := c.GroupEntriesByTask(pending)
	summedExpected := c.SumEntriesGroup(grouped)

	summed, err := c.SendToPlanfix()
	assert.NoError(t, err)
	assert.Equal(t, summedExpected, summed)
}
