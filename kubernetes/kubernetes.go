package kubernetes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/giantswarm/benchmarktosheet/config"
	sheets "google.golang.org/api/sheets/v4"
)

type Result struct {
	TestNumber string   `json:"test_number"`
	TestDesc   string   `json:"test_desc"`
	Type       string   `json:"type"`
	TestInfo   []string `json:"test_info"`
	Status     string   `json:"status"`
}
type Test struct {
	Section string   `json:"section"`
	Pass    int      `json:"pass"`
	Fail    int      `json:"fail"`
	Warn    int      `json:"warn"`
	Desc    string   `json:"desc"`
	Results []Result `json:"results"`
}
type KubeBenchResult struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	Text      string `json:"text"`
	NodeType  string `json:"node_type"`
	Tests     []Test `json:"tests"`
	TotalPass int    `json:"total_pass"`
	TotalFail int    `json:"total_fail"`
	TotalWarn int    `json:"total_warn"`
}

func CreateSheet(srv *sheets.Service, spreadsheetID, name string) (string, int64, error) {
	t := time.Now().Local()
	aRequest := sheets.AddSheetRequest{
		Properties: &sheets.SheetProperties{
			Title: fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()),
		},
	}
	sRequests := []*sheets.Request{}
	sRequests = append(sRequests, &sheets.Request{
		AddSheet: &aRequest,
	})
	bRequest := sheets.BatchUpdateSpreadsheetRequest{
		Requests: sRequests,
	}
	resp, err := srv.Spreadsheets.BatchUpdate(spreadsheetID, &bRequest).Do()

	if err != nil {
		return "", 0, err
	}
	return resp.Replies[0].AddSheet.Properties.Title, resp.Replies[0].AddSheet.Properties.SheetId, nil
}

func ParseKubeBench(filepath string) (*KubeBenchResult, error) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var k KubeBenchResult
	err = json.Unmarshal(b, &k)

	if err != nil {
		return nil, err
	}

	return &k, nil

}
func insertTitle(srv *sheets.Service, spreadsheetID string, sheetId int64, sheetName string, title string, index int64) error {
	v := []interface{}{
		title,
	}
	vv := [][]interface{}{v}
	valueRange := sheets.ValueRange{
		Values: vv,
	}
	_, err := srv.Spreadsheets.Values.Update(spreadsheetID, fmt.Sprintf("%v!A%v:A%v", sheetName, index, index), &valueRange).ValueInputOption("RAW").Do()
	if err != nil {
		return err
	}

	bRequest := sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			&sheets.Request{
				MergeCells: &sheets.MergeCellsRequest{
					MergeType: "MERGE_ALL",
					Range: &sheets.GridRange{
						SheetId:          sheetId,
						StartColumnIndex: 0,
						StartRowIndex:    index - 1,
						EndColumnIndex:   4,
						EndRowIndex:      index,
					},
				},
			},
			&sheets.Request{
				RepeatCell: &sheets.RepeatCellRequest{
					Range: &sheets.GridRange{
						SheetId:          sheetId,
						StartColumnIndex: 0,
						StartRowIndex:    index - 1,
						EndColumnIndex:   4,
						EndRowIndex:      index,
					},
					Cell: &sheets.CellData{
						UserEnteredFormat: &sheets.CellFormat{
							TextFormat: &sheets.TextFormat{
								Bold:     true,
								FontSize: 14,
							},
						},
					},
					Fields: "userEnteredFormat(textFormat)",
				},
			},
		},
	}
	_, err = srv.Spreadsheets.BatchUpdate(spreadsheetID, &bRequest).Do()
	if err != nil {
		return err
	}

	return nil
}

func InsertTotals(srv *sheets.Service, spreadsheetID string, sheetName string, result *KubeBenchResult, index int) error {

	v := []interface{}{
		fmt.Sprintf("%v FAIL", result.TotalFail),
		fmt.Sprintf("%v WARN", result.TotalWarn),
		fmt.Sprintf("%v PASS", result.TotalPass),
	}
	vv := [][]interface{}{v}
	valueRange := sheets.ValueRange{
		Values: vv,
	}
	_, err := srv.Spreadsheets.Values.Update(spreadsheetID, fmt.Sprintf("%v!B%v:D%v", sheetName, index, index), &valueRange).ValueInputOption("RAW").Do()
	if err != nil {
		return err
	}
	return nil
}
func InsertResult(srv *sheets.Service, spreadsheetID string, sheetId int64, sheetName string, report config.Report, startRow int) (int, error) {
	log.Println("Parse report:", report.Name)
	result, err := ParseKubeBench(report.Path)
	if err != nil {
		return 0, err
	}
	err = insertTitle(srv, spreadsheetID, sheetId, sheetName, report.Name, int64(startRow+1))
	if err != nil {
		return 0, err
	}

	err = InsertTotals(srv, spreadsheetID, sheetName, result, startRow+2)
	if err != nil {
		return 0, err
	}

	ARange, BRange := startRow+3, startRow+3
	for _, test := range result.Tests {
		BRange = BRange + len(test.Results) + 1
		section := []interface{}{
			"INFO",
			test.Section,
		}

		vv := [][]interface{}{section}
		for _, r := range test.Results {
			vv = append(vv, []interface{}{
				r.Status,
				fmt.Sprintf("%v %v", r.TestNumber, r.TestDesc),
			})
		}

		valueRange := sheets.ValueRange{
			Values: vv,
		}
		log.Printf("Inserting Section %v in %v!A%v:B%v", test.Section, sheetName, ARange, BRange)
		_, err := srv.Spreadsheets.Values.Update(spreadsheetID, fmt.Sprintf("%v!A%v:B%v", sheetName, ARange, BRange), &valueRange).ValueInputOption("RAW").Do()
		if err != nil {
			return 0, err
		}
		ARange = BRange + 1
	}
	format := map[string]*sheets.Color{
		"PASS": &sheets.Color{
			Red:   0.72,
			Green: 0.88,
			Blue:  0.8,
		},
		"WARN": &sheets.Color{
			Red:   0.99,
			Green: 0.91,
			Blue:  0.70,
		},
		"FAIL": &sheets.Color{
			Red:   0.96,
			Green: 0.78,
			Blue:  0.76,
		},
	}

	sRequests := []*sheets.Request{}

	for status, color := range format {

		sRequests = append(sRequests, &sheets.Request{
			AddConditionalFormatRule: &sheets.AddConditionalFormatRuleRequest{
				Index: 0,
				Rule: &sheets.ConditionalFormatRule{
					BooleanRule: &sheets.BooleanRule{
						Condition: &sheets.BooleanCondition{
							Type: "TEXT_CONTAINS",
							Values: []*sheets.ConditionValue{
								&sheets.ConditionValue{UserEnteredValue: status},
							},
						},
						Format: &sheets.CellFormat{
							BackgroundColor: color,
						},
					},
					Ranges: []*sheets.GridRange{
						&sheets.GridRange{
							SheetId:          sheetId,
							StartColumnIndex: 0,
							StartRowIndex:    0,
							EndColumnIndex:   4,
							EndRowIndex:      997,
						},
					},
				},
			},
		})
	}

	bRequest := sheets.BatchUpdateSpreadsheetRequest{
		Requests: sRequests,
	}
	_, err = srv.Spreadsheets.BatchUpdate(spreadsheetID, &bRequest).Do()

	if err != nil {
		return 0, err
	}
	return ARange + 1, nil
}
