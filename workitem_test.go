// +build integration

package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"

	"github.com/jinzhu/gorm"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	dbhost := os.Getenv("ALMIGHTY_DB_HOST")
	if "" == dbhost {
		panic("The environment variable ALMIGHTY_DB_HOST is not specified or empty.")
	}
	var err error
	db, err = gorm.Open("postgres", fmt.Sprintf("host=%s user=postgres password=mysecretpassword sslmode=disable", dbhost))
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	defer db.Close()
	// Migrate the schema
	migration.Perform(db)
	m.Run()
}

func TestGetWorkItem(t *testing.T) {
	ts := models.NewGormTransactionSupport(db)
	repo := models.NewWorkItemRepository(ts)
	controller := WorkitemController{ts: ts, wiRepository: repo}
	payload := app.CreateWorkitemPayload{
		Type: "1",
		Fields: map[string]interface{}{
			"system.title": "foobar",
			"system.owner": "aslak",
			"system.state": "done"},
	}
	fmt.Println(payload.Fields)

	_, result := test.CreateWorkitemCreated(t, nil, nil, &controller, &payload)

	_, wi := test.ShowWorkitemOK(t, nil, nil, &controller, result.ID)

	if wi == nil {
		t.Fatalf("Work Item '%s' not present", result.ID)
	}

	if wi.ID != result.ID {
		t.Errorf("Id should be %s, but is %s", result.ID, wi.ID)
	}

	wi.Fields["system.owner"] = "thomas"
	payload2 := app.UpdateWorkitemPayload{
		Type:    wi.Type,
		Version: wi.Version,
		Fields:  wi.Fields,
	}
	_, updated := test.UpdateWorkitemOK(t, nil, nil, &controller, wi.ID, &payload2)
	if updated.Version != result.Version+1 {
		t.Errorf("expected version %d, but got %d", result.Version+1, updated.Version)
	}
	if updated.ID != result.ID {
		t.Errorf("id has changed from %s to %s", result.ID, updated.ID)
	}
	if updated.Fields["system.owner"] != "thomas" {
		t.Errorf("expected owner %s, but got %s", "thomas", updated.Fields["system.owner"])
	}

	test.DeleteWorkitemOK(t, nil, nil, &controller, result.ID)
}

func TestCreateWI(t *testing.T) {
	ts := models.NewGormTransactionSupport(db)
	repo := models.NewWorkItemRepository(ts)
	controller := WorkitemController{ts: ts, wiRepository: repo}
	payload := app.CreateWorkitemPayload{
		Type: "1",
		Fields: map[string]interface{}{
			"system.title": "some name ",
			"system.owner": "tmaeder",
			"system.state": "open",
		},
	}

	_, created := test.CreateWorkitemCreated(t, nil, nil, &controller, &payload)
	if created.ID == "" {
		t.Error("no id")
	}
}

func TestListByFields(t *testing.T) {
	ts := models.NewGormTransactionSupport(db)
	repo := models.NewWorkItemRepository(ts)
	controller := WorkitemController{ts: ts, wiRepository: repo}
	payload := app.CreateWorkitemPayload{
		Type: "1",
		Fields: map[string]interface{}{
			"system.title": "ListByName Name",
			"system.owner": "aslak",
			"system.state": "done",
		},
	}

	_, wi := test.CreateWorkitemCreated(t, nil, nil, &controller, &payload)

	/*
		filter := "{\"Name\":\"ListByName Name\"}"
		page := "1,1"
		_, result := test.ListWorkitemOK(t, nil, nil, &controller, &filter, &page)

		if result == nil {
			t.Errorf("nil result")
		}
	*/
	if len(result) != 1 {
		t.Errorf("unexpected length, is %d but should be %d", 1, len(result))
	}

	filter = "{\"system.owner\":\"aslak\"}"
	_, result = test.ListWorkitemOK(t, nil, nil, &controller, &filter, &page)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result) != 1 {
		t.Errorf("unexpected length, is %d but should be %d", 1, len(result))
	}

	test.DeleteWorkitemOK(t, nil, nil, &controller, wi.ID)
}
