package database

import (
	"strings"
	"fmt"
	"database/sql"
	"modernc.org/ql"
	"encoding/json"
	"encoding/base64"

	"taylor/lib/structs"
	//"taylor/lib/util"
)

type Store struct {
	db *sql.DB
}

func (s *Store) init() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS jobs (
		id STRING
		,identifier STRING
		,status INT
		,ts BIGINT
		,agent_name STRING
		,driver STRING
		,driver_config STRING
		,update_handlers STRING
		,restrict STRING
		,priority INT
		,progress FLOAT
	);
	`)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func Open(dbPath string) (*Store, error) {

	ql.RegisterDriver()

	store := &Store{}

	dbh, err := sql.Open("ql", dbPath)
	if err != nil {
		dbh.Close()
		return nil, err
	}

	store.db = dbh

	err = store.init()
	if err != nil {
		dbh.Close()
		return nil, err
	}

	return store, nil
}

func encodeData(data interface{}) (string, error) {
	js, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(js), nil
}

func decodeData(data string) (interface{}, error) {
	hsJson, err := base64.RawStdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	var decoded interface{}
	err = json.Unmarshal(hsJson, &decoded)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func (s *Store) InsertJob(job *structs.Job) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}

	driverConfig, _ := encodeData(job.DriverConfig)
	updateHandlers, _ := encodeData(job.UpdateHandlers)
	restrict, _ := encodeData(job.Restrict)

	query := fmt.Sprintf(`
	INSERT INTO jobs (
		id,
		identifier,
		status,
		ts,
		agent_name,
		driver,
		driver_config,
		update_handlers,
		restrict,
		priority,
		progress
	) VALUES ("%s", "%s", %d, %d, "%s", "%s", "%s", "%s", "%s", %d, %f)
	`,
		job.Id,
		job.Identifier,
		job.Status,
		job.Timestamp,
		job.AgentName,
		job.Driver,
		string(driverConfig),
		string(updateHandlers),
		string(restrict),
		job.Priority,
		job.Progress,
	)

	_, err = tx.Exec(query)
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return 0, nil
}

func (s *Store) IterQuery(query string, fun func (job *structs.Job)) error {
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var job structs.Job
		
		var encodedDriverConfig string
		var encodedUpdateHandlers string
		var encodedRestrict string

		err := rows.Scan(
			&job.Id,
			&job.Identifier,
			&job.Status,
			&job.Timestamp,
			&job.AgentName,
			&job.Driver,
			&encodedDriverConfig,
			&encodedUpdateHandlers,
			&encodedRestrict,
			&job.Priority,
			&job.Progress,
		)
		if err != nil {
			return err
		}

		decodedDriverConfig, _ := decodeData(encodedDriverConfig)
		driverConfig, _ := decodedDriverConfig.(map[string]interface{})

		decodedUpdateHandlers, _ := decodeData(encodedUpdateHandlers)
		updateHandlersTmp, _ := decodedUpdateHandlers.([]interface{})

		// I know. I am not proud of this but I am too lazy now and want to make progress. 
		updateHandlers := make([]structs.UpdateHandler, len(updateHandlersTmp))
		for i, handlerTmp := range updateHandlersTmp {
			handler, _ := handlerTmp.(map[string]interface{})	
			handlerType, _ := handler["type"].(string)
			handlerOnTmps, _ := handler["on"].([]interface{})
			handlerOn := make([]string, len(handlerOnTmps))
			for j, onTmp := range handlerOnTmps {
				on, _ := onTmp.(string)
				handlerOn[j] = on
			}

			var handlerCfgTmp map[string]interface{}
			handlerCfgTmp, ok := handler["config"].(map[string]interface{})
			if ok == false || handlerCfgTmp == nil {
				handlerCfgTmp = make(map[string]interface{}, 0)
			}

			updateHandlers[i] = structs.UpdateHandler{
				Type:		handlerType,
				OnEventList:	handlerOn,
				Config:		handlerCfgTmp,
			}
		}

		decodedRestrict, _ := decodeData(encodedRestrict)
		restrictTmp, _ := decodedRestrict.([]interface{})
		restrict := make([]string, len(restrictTmp))

		for i, tmp := range(restrictTmp) {
			r, _ := tmp.(string)
			restrict[i] = r
		}

		job.DriverConfig = driverConfig
		job.UpdateHandlers = updateHandlers
		job.Restrict = restrict

		fun(&job)
	}

	return nil
}

func (s *Store) CollectQuery(query string) ([]*structs.Job, error) {
	jobs := make([]*structs.Job, 0)
	error := s.IterQuery(query, func (job *structs.Job) {
		jobs = append(jobs, job)
	})
	if error != nil {
		return nil, error
	}
	return jobs, nil
}

func (s *Store) JobById(id string) (*structs.Job, error) {
	q := fmt.Sprintf("SELECT * FROM jobs WHERE id = \"%s\";", id)

	jobs, err := s.CollectQuery(q)
	if err != nil {
		return nil, err
	}
	
	if len(jobs) == 0 {
		return nil, nil
	}

	return jobs[0], nil
}

func (s *Store) AllJobs(limit uint) ([]*structs.Job, error) {
	var q strings.Builder

	q.WriteString(fmt.Sprintf("SELECT * FROM jobs ORDER BY ts ASC"))
	if limit > 0 {
		q.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	}

	return s.CollectQuery(q.String())
}

func (s *Store) JobsFromNodeWithStatus(nodeName string, status structs.JobStatus) ([]*structs.Job, error) {

	query := fmt.Sprintf("SELECT * FROM jobs WHERE agent_name = \"%s\" AND status = %d ORDER BY ts ASC", nodeName, int(status))

	return s.CollectQuery(query)
}

func (s *Store) JobsWithStatus(status structs.JobStatus, limit uint) ([]*structs.Job, error) {
	var q strings.Builder

	q.WriteString(fmt.Sprintf("SELECT * FROM jobs WHERE status = %d ORDER BY ts ASC", int(status)))
	if limit > 0 {
		q.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	}

	return s.CollectQuery(q.String())
}

func (s *Store) UpdateJobAgentName(id string, agentName string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	q := fmt.Sprintf("UPDATE jobs SET agent_name = \"%s\" WHERE id = \"%s\"", agentName, id)

	_, err = tx.Exec(q)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) UpdateJobStatus(id string, status structs.JobStatus) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	q := fmt.Sprintf("UPDATE jobs SET status = %d WHERE id = \"%s\"", int(status), id)

	_, err = tx.Exec(q)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) UpdateJobProgress(id string, progress float32) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	q := fmt.Sprintf("UPDATE jobs SET progress = %f WHERE id = \"%s\"", progress, id)

	_, err = tx.Exec(q)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
