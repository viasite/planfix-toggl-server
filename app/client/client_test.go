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

type planfixRequestStruct struct {
	XMLName xml.Name `xml:"request"`
	Method  string   `xml:"method,attr"`
	Account string   `xml:"account"`
	Sid     string   `xml:"sid"`
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

func newClient() TogglClient {
	cfg := config.Config{}
	ms := NewMockedServer([]string{""})
	api := planfix.New(ms.URL, "apiKey", "account", "user", "password")
	api.Sid = "123"

	return TogglClient{
		Session:    toggl.OpenSession(cfg.TogglAPIToken),
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
