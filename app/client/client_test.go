package client

import (
	"testing"
	"log"
	"github.com/viasite/planfix-toggl-server/app/config"
	"github.com/popstas/planfix-go/planfix"
	"net/http/httptest"
	"net/http"
	"io/ioutil"
	"encoding/xml"
	"github.com/popstas/go-toggl"
	"reflect"
	"time"
	"bytes"
	"strings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	equals := reflect.DeepEqual(summed, expected)
	assert.Equal(t, equals, true)
}

func TestTogglClient_GroupEntriesByTask(t *testing.T) {
	c := newClient()
	entries := getTestEntries()
	expected := getTestGroupedEntries()

	grouped := c.GroupEntriesByTask(entries)
	equals := reflect.DeepEqual(grouped, expected)
	assert.Equal(t, equals, true)
}

func TestTogglClient_GroupEntriesByTask_empty(t *testing.T) {
	c := newClient()
	entries := []TogglPlanfixEntry{}
	expected := map[int][]TogglPlanfixEntry{}

	grouped := c.GroupEntriesByTask(entries)
	equals := reflect.DeepEqual(grouped, expected)
	assert.Equal(t, equals, true)
}

func TestTogglClient_sendEntries_dryRun(t *testing.T) {
	c := newClient()
	c.Config.DryRun = true
	now := time.Now()
	entries := []TogglPlanfixEntry{
		{
			toggl.DetailedTimeEntry{
				Duration:    60000,
				Start:       &now,
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
	assert.Equal(t, strings.Contains(output.String(), "[DEBUG] sending [project] description (3)"), true)
	assert.Equal(t, strings.Contains(output.String(), "[DEBUG] dry-run"), true)
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
	assert.Equal(t, togglUserID, 123)
}

func TestTogglClient_GetPlanfixUserID(t *testing.T) {
	c := newClient()
	ms := NewMockedServer([]string{
		fixtureFromFile("user.get.xml"),
		//fixtureFromFile("error.xml"),
	})
	c.PlanfixAPI.URL = ms.URL

	planfixUserID := c.GetPlanfixUserID()
	assert.Equal(t, planfixUserID, 9230)
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
	assert.Equal(t, entries, expected)
}

func TestTogglClient_GetPendingEntries(t *testing.T) {
	c := newClient()
	sess := &MockedTogglSession{}
	c.Session = sess

	since := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	until := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	report := toggl.DetailedReport{Data: []toggl.DetailedTimeEntry{
		// will be filtered by sent tag
		{
			ID:          1,
			Project:     "project1",
			Description: "description1",
			Tags:        []string{"12345", "sent"},
		},
		{
			ID:          2,
			Project:     "project1",
			Description: "description1",
			Tags:        []string{"12345"},
		},
		// will be filtered by taskID tag
		{
			ID:          3,
			Project:     "project1",
			Description: "description1",
			Tags:        []string{},
		},
		{
			ID:          4,
			Project:     "project1",
			Description: "description1",
			Tags:        []string{"12345"},
		},
	}}

	sess.On("GetDetailedReport", c.Config.TogglWorkspaceID, since, until, 1).Return(report, nil)

	entries, _ := c.GetPendingEntries()

	expected := []TogglPlanfixEntry{
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
	assert.Equal(t, entries, expected)
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
