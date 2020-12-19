package dagrun

import (
	"fmt"
	dagtable "goflow/internal/dag/sql/dag"
	"goflow/internal/database"
	"goflow/internal/testutils"
	"path"
	"testing"
	"time"
)

var sqlClient *database.SQLClient
var tableClient *TableClient

var databaseFile = path.Join(testutils.GetTestFolder(), "test.sqlite3")
var testDagRow = dagtable.NewRow(0, "dag_num_1", "default", "v1", "/my/path", "json")

func setUpDagTable() {
	dagTableClient := dagtable.NewTableClient(sqlClient)
	dagTableClient.CreateTable()
	dagTableClient.UpsertDag(testDagRow)
}

func TestMain(m *testing.M) {
	sqlClient = database.NewSQLiteClient(databaseFile)
	tableClient = NewTableClient(sqlClient)
	m.Run()
}

func TestCreateDagRunTable(t *testing.T) {
	defer database.PurgeDB(sqlClient)
	tableClient.CreateTable()
	found := false
	for _, table := range sqlClient.Tables() {
		if table == tableName {
			found = true
		}
	}
	if !found {
		t.Errorf("Did not find table %s in tables", tableName)
	}
}

func insertDaysOfRuns(n int) []Row {
	if n > 31 {
		panic("Can only go up to 31")
	}
	insertedRows := make([]Row, 0, n)
	for i := 0; i < n; i++ {
		timeStr := fmt.Sprintf("2019-01-%02d", i+1)
		executionTime, err := time.Parse("2006-01-02", timeStr)
		if err != nil {
			panic(err)
		}

		newRow := NewRow(testDagRow.ID, "Running", executionTime)
		insertedRows = append(insertedRows, newRow)
		sqlClient.Insert(tableName, newRow.columnar())
	}
	return insertedRows
}

func TestGetLastNDagRuns(t *testing.T) {
	defer database.PurgeDB(sqlClient)
	sqlClient.CreateTable(tableClient.tableDef)

	const insertedDays = 31
	expectedRows := insertDaysOfRuns(insertedDays)

	const expectedRowCount = 5
	foundRows := tableClient.GetLastNRunsForDagID(testDagRow.ID, expectedRowCount)

	length := len(foundRows)
	if length != expectedRowCount {
		t.Errorf("Expected %d rows, found %d", expectedRowCount, length)
	}
	for i, row := range foundRows {
		expectedRow := expectedRows[i+insertedDays-expectedRowCount]
		if row != expectedRow {
			t.Errorf("expected row %s, found row %s", expectedRow, row)
			panic("test")
		}
	}

}

func TestUpsertDagRun(t *testing.T) {

}
