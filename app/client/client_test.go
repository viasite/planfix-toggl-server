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
)

var output bytes.Buffer

func assert(t *testing.T, data interface{}, expected interface{}) {
	if data != expected {
		t.Errorf("Expected %v, got, %v", expected, data)
	}
}
func expectError(t *testing.T, err error, msg string) {
	if err == nil {
		t.Errorf("Expected error, got success %v", msg)
	}
}
func expectSuccess(t *testing.T, err error, msg string) {
	if err != nil {
		t.Errorf("Expected success, got %v %v", err, msg)
	}
}

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
	TogglSession
}

func (s *MockedTogglSession) GetAccount() (toggl.Account, error) {
	return toggl.Account{Data: struct {
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
	}{ID: 123}}, nil
}

func (s *MockedTogglSession) GetDetailedReport(workspace int, since, until string, page int) (toggl.DetailedReport, error) {
	return toggl.DetailedReport{Data:[]toggl.DetailedTimeEntry{
		{
			ID: 1,
			Project: "project1",
			Description: "description1",
			Tags: []string{"12345", "sent"},
		},
	}}, nil
}

func newClient() TogglClient {
	cfg := config.Config{
		TogglSentTag:"sent",
	}
	api := planfix.New("", "apiKey", "account", "user", "password")
	api.Sid = "123"

	sess := MockedTogglSession{}
	return TogglClient{
		Session:    &sess,
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
	assert(t, equals, true)
}

func TestTogglClient_GroupEntriesByTask(t *testing.T) {
	c := newClient()
	entries := getTestEntries()
	expected := getTestGroupedEntries()

	grouped := c.GroupEntriesByTask(entries)
	equals := reflect.DeepEqual(grouped, expected)
	assert(t, equals, true)
}

func TestTogglClient_GroupEntriesByTask_empty(t *testing.T) {
	c := newClient()
	entries := []TogglPlanfixEntry{}
	expected := map[int][]TogglPlanfixEntry{}

	grouped := c.GroupEntriesByTask(entries)
	equals := reflect.DeepEqual(grouped, expected)
	assert(t, equals, true)
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
	assert(t, strings.Contains(output.String(), "[DEBUG] sending [project] description (3)"), true)
	assert(t, strings.Contains(output.String(), "[DEBUG] dry-run"), true)
}

func TestTogglClient_GetTogglUserID(t *testing.T) {
	c := newClient()
	togglUserID := c.GetTogglUserID()
	assert(t, togglUserID, 123)
}

func TestTogglClient_GetPlanfixUserID(t *testing.T) {
	c := newClient()
	ms := NewMockedServer([]string{
		fixtureFromFile("user.get.xml"),
		//fixtureFromFile("error.xml"),
	})
	c.PlanfixAPI.URL = ms.URL

	planfixUserID := c.GetPlanfixUserID()
	assert(t, planfixUserID, 9230)
}

func TestTogglClient_GetEntries(t *testing.T) {
	c := newClient()
	entries, err := c.GetEntries(
		c.Config.TogglWorkspaceID,
		time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
		time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
	)

	report, err := c.Session.GetDetailedReport(
		c.Config.TogglWorkspaceID,
		time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
		time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
		1,
	)

	expected := []TogglPlanfixEntry{
		{
			DetailedTimeEntry: report.Data[0],
			Planfix: PlanfixEntryData{GroupCount:1, Sent:true, TaskID:12345},
		},
	}
	equals := reflect.DeepEqual(entries, expected)
	expectSuccess(t, err, "TestTogglClient_GetEntries")
	assert(t, equals, true)
}