package database

import (
	//"errors"
	"strings"
	"fmt"
	"database/sql"
	"modernc.org/ql"

	"taylor/lib/structs"
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

func (s *Store) InsertJob(job *structs.Job) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf(`
	INSERT INTO jobs (id, identifier, status, ts) VALUES ("%s", "%s", %d, %d);
	`, job.Id, job.Identifier, job.Status, job.Timestamp)

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

		err := rows.Scan(&job.Id, &job.Identifier, &job.Status, &job.Timestamp)
		if err != nil {
			return err
		}

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

func (s *Store) JobsWithStatus(status structs.JobStatus, limit uint) ([]*structs.Job, error) {
	var q strings.Builder

	q.WriteString(fmt.Sprintf("SELECT * FROM jobs WHERE status = %d ORDER BY ts ASC", int(status)))
	if limit > 0 {
		q.WriteString(fmt.Sprintf(" LIMIT %d", limit))
	}

	return s.CollectQuery(q.String())
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
