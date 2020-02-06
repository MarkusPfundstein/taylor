package handlers

import (
	"net/http"
	"bytes"
	"os"
	"io/ioutil"
	"encoding/json"
	"errors"
	"time"
	"fmt"
	"taylor/lib/util"
	"taylor/lib/structs"
)

func ExecWebhook(config map[string]interface{}, job *structs.Job, eventName string, progress float32, payload string) error {
	method, err := util.GetString(config, "method", "POST")
	if err != nil {
		return err
	}
	url, err := util.GetString(config, "url", "")
	if err != nil {
		return err
	}
	if url == "" {
		return errors.New("No url in webhook config")
	}
	fullJob, err := util.GetBool(config, "full_job_description", false)
	if err != nil {
		return err
	}

	go func() {
		client := http.Client{
			Timeout: 5 * time.Second,
		}
		switch (method) {
		case "GET":
			resp, err := client.Get(url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[webhook] %v\n", err)
				return
			}
			defer resp.Body.Close()
		case "POST":
			data := make(map[string]interface{}, 0)
			if fullJob == true {
				data["job"] = *job
			} else {
				data["job_id"] = job.Id
			}
			data["eventName"] = eventName
			data["progress"] = progress
			data["message"] = payload

			json, _ := json.Marshal(data)
			resp, err := client.Post(url, "application/json", bytes.NewBuffer(json))
			if err != nil {
				fmt.Fprintf(os.Stderr, "[webhook] %v\n", err)
				return
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[webhook] %v\n", err)
				return
			}
			fmt.Printf("%s > %s\n", url, string(body))
		}
	}()
	return nil
}

