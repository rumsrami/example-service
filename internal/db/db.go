package db

import (
	"context"
	"fmt"
	"log"

	"github.com/pkg/errors"
)

type PartitionKey struct {
	DriverName string
	Week       int
}

type SortKey struct {
	Day       int
	StartHour int
}

type Task struct {
	Ops       string
	StartHour int
	Duration  int
}

type Database struct {
	quitCh   chan chan struct{}
	actionCh chan func()
	Schedule map[PartitionKey]map[SortKey]Task
}

func NewPartitionKey(driverName string, week int) PartitionKey {
	return PartitionKey{
		DriverName: driverName,
		Week:       week,
	}
}

func NewSortKey(day, startHr int) SortKey {
	return SortKey{
		Day:       day,
		StartHour: startHr,
	}
}

func NewTask(operation string, startHr, duration int) Task {
	return Task{
		Ops:       operation,
		StartHour: startHr,
		Duration:  duration,
	}
}

func NewDatabase() *Database {
	return &Database{
		Schedule: make(map[PartitionKey]map[SortKey]Task),
		quitCh:   make(chan chan struct{}),
		actionCh: make(chan func(), 1000),
	}
}

func (d *Database) Run() error{
	defer func() {
		log.Println("Database closed")
	}()

	for {
		select {
		case f := <-d.actionCh:
			log.Println("An action came")
			f()
		case q := <-d.quitCh:
			close(q)
			return nil
		}
	}
}

func (d *Database) Stop() {
	q := make(chan struct{})
	d.quitCh <- q
	// This blocks until Run() closes q and returns
	<-q
}

func (d *Database) CreateTask(ctx context.Context, task Task, partitionKey PartitionKey, sortKey SortKey) error {
	var e = make(chan error, 100) // Load 'Requests/sec: 19187.9399'
	d.actionCh <- func() {
		if dbTaskSortKeyMap, ok := d.Schedule[partitionKey]; ok {
			if _, ok := dbTaskSortKeyMap[sortKey]; ok {
				e <- errors.New("Task already exists")
				return
			} else {
				dbTaskSortKeyMap[sortKey] = task
				fmt.Println(dbTaskSortKeyMap[sortKey], task)
				e <- nil
				return
			}
		} else {
			newTaskSortKeyMap := make(map[SortKey]Task)
			newTaskSortKeyMap[sortKey] = task
			d.Schedule[partitionKey] = newTaskSortKeyMap
			e <- nil
			return
		}
	}
	select {
	case err := <-e:
		return err
	}
}

func (d *Database) DeleteTask(ctx context.Context, partitionKey PartitionKey, sortKey SortKey) error {
	e := make(chan error)
	d.actionCh <- func() {
		if dbTaskSortKeyMap, ok := d.Schedule[partitionKey]; ok {
			if _, ok := dbTaskSortKeyMap[sortKey]; ok {
				delete(d.Schedule[partitionKey], sortKey)
				e <- nil
				return
			}
			e <- errors.New("Task doesnt exists")
			return
		}
		e <- errors.New("Task doesnt exists")
		return
	}
	select {
	case err := <-e:
		return err
	}
}

func (d *Database) ReadTask(ctx context.Context, partitionKey PartitionKey, sortKey SortKey) (Task, error) {
	e := make(chan error)
	t := make(chan Task)
	d.actionCh <- func() {
		if dbTaskSortKeyMap, ok := d.Schedule[partitionKey]; ok {
			if task, ok := dbTaskSortKeyMap[sortKey]; ok {
				t <- task
			}
			e <- errors.New("Task doesnt exists")
		}
		e <- errors.New("Task doesnt exists")
	}
	select {
	case err := <-e:
		return Task{}, err
	case task := <-t:
		return task, nil
	}
}

func (d *Database) ReadSchedule(ctx context.Context, partitionKey PartitionKey) (map[SortKey]Task, error) {
	e := make(chan error, 100)
	s := make(chan map[SortKey]Task, 10) // Load Requests/sec: 16541.7226
	d.actionCh <- func() {
		if dbTaskSortKeyMap, ok := d.Schedule[partitionKey]; ok {
			s <- dbTaskSortKeyMap
			return
		}
		// change this to have the http status error to signal back to the caller, now it defaults to 500 internal server, should be 404
		e <- errors.New("Schedule doesnt exist doesnt exists")
		return
	}
	select {
	case err := <-e:
		return map[SortKey]Task{}, err
	case schedule := <-s:
		return schedule, nil
	}
}
