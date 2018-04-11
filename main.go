package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/giantswarm/benchmarktosheet/config"
	"github.com/giantswarm/benchmarktosheet/kubernetes"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

func main() {

	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "config",
		},
	}

	app.Action = func(c *cli.Context) error {
		confPath := c.String("config")
		if confPath == "" {
			log.Fatal("--config must be set")
		}
		conf := config.Config{}
		f, err := ioutil.ReadFile(confPath)
		err = json.Unmarshal(f, &conf)

		if err != nil {
			log.Fatal(err)
		}

		ctx := context.Background()

		b, err := ioutil.ReadFile("client_secret.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
		}

		// If modifying these scopes, delete your previously saved credentials
		// at ~/.credentials/sheets.googleapis.com-go-quickstart.json
		config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
		if err != nil {
			log.Fatalf("Unable to parse client secret file to config: %v", err)
		}
		client := getClient(ctx, config)

		srv, err := sheets.New(client)
		if err != nil {
			log.Fatalf("Unable to retrieve Sheets Client %v", err)
		}

		// Prints the names and majors of students in a sample spreadsheet:
		// https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit
		spreadsheetId := "1UQAtOJuJXchRE9EZ6_460jKHON3FtjltU2mWnkqCRX4"
		sheetName, sheetId, err := kubernetes.CreateSheet(srv, spreadsheetId, time.Now().Local().Format("2006-12-1"))
		if err != nil {
			log.Fatalf("Unable to retrieve Sheets Client %v", err)
		}

		startRow := 0
		for _, report := range conf.Reports {

			startRow, err = kubernetes.InsertResult(srv, spreadsheetId, sheetId, sheetName, report, startRow)
			if err != nil {
				log.Fatalf("Unable to retrieve data from sheet. %v", err)
			}
		}

		return nil
	}
	app.Run(os.Args)
}
