package main

import (
	"fmt"
	"time"
)

// Приложение эмулирует получение и обработку неких тасков. Пытается и получать, и обрабатывать в многопоточном режиме.
// После обработки тасков в течении 3 секунд приложение должно выводить накопленные к этому моменту успешные таски и отдельно ошибки обработки тасков.

// ЗАДАНИЕ: сделать из плохого кода хороший и рабочий - as best as you can.
// Важно сохранить логику появления ошибочных тасков.
// Важно оставить асинхронные генерацию и обработку тасков.
// Сделать правильную мультипоточность обработки заданий.
// Обновленный код отправить через pull-request в github
// Как видите, никаких привязок к внешним сервисам нет - полный карт-бланш на модификацию кода.

const (
	// available statuses
	TStatusOk   = 0
	TStatusFail = 1

	// settings of times
	WaitingTime            = 1 * time.Second
	DelayWorkTasks         = 150 * time.Millisecond
	ValidatedTimeBeforeNow = -20 * time.Second
)

// A Ttype represents a meaninglessness of our life
type Ttype struct {
	id         int64
	cT         string // время создания
	fT         string // время выполнения
	taskStatus int
	taskResult []byte
}

// Validate task that it's ok
//
// Not ok if:
// - cannot parsed
// - timeout of N seconds
func (t *Ttype) taskIsOk() bool {
	tt, err := time.Parse(time.RFC3339, t.cT)
	if err != nil {
		return false
	}

	if !tt.After(time.Now().Add(ValidatedTimeBeforeNow)) {
		return false
	}

	return true
}

type TProcess struct {
	// main chan with tasks
	taskChan chan *Ttype

	// sort chan with Done tasks
	doneChan chan *Ttype
	// sort chan with Undone tasks
	undoneChan chan error

	// finish work chan
	finish chan struct{}

	// reports
	results map[int64]*Ttype
	errors  []error
}

// Constructor of Process
func createProcess() *TProcess {
	process := TProcess{
		taskChan:   make(chan *Ttype, 10),
		doneChan:   make(chan *Ttype, 10),
		undoneChan: make(chan error, 10),
		finish:     make(chan struct{}),
		results:    map[int64]*Ttype{},
		errors:     []error{},
	}

	return &process
}

// Start method of creation tasks
func (p *TProcess) taskCreturer() {
	for {
		select {
		case <-p.finish:
			// if finish - than finish
			close(p.taskChan)
			return
		default:
			t := time.Now()
			ft := t.Format(time.RFC3339)
			if t.UnixMicro()%2 > 0 { // вот такое условие появления ошибочных тасков
				ft = "Some error occured"
			}

			// передаем таск на выполнение
			p.taskChan <- &Ttype{
				id: t.UnixMicro(),
				cT: ft,
			}
		}
	}
}

// Execute some business logic on task
func (p *TProcess) workWithTask(t *Ttype) *Ttype {
	if !t.taskIsOk() {
		t.taskResult = []byte("something went wrong")
		t.taskStatus = TStatusFail
	} else {
		t.taskResult = []byte("task has been successed")
		t.taskStatus = TStatusOk
	}

	t.fT = time.Now().Format(time.RFC3339Nano)

	// time.Sleep(DelayWorkTasks)

	return t
}

// Send task to channels by status of work
func (p *TProcess) taskSorter(t *Ttype) {
	if t.taskStatus == TStatusOk {
		p.doneChan <- t
	} else {
		p.undoneChan <- fmt.Errorf("task id %d time %s, error %s", t.id, t.cT, t.taskResult)
	}
}

// Get task from main channel and process it
func (p *TProcess) processTasks() {
	for {
		select {
		case <-p.finish:
			return
		case task, ok := <-p.taskChan:
			if !ok {
				return
			}

			go p.subProcess(task)
		}
	}
}

// Process task async
func (p *TProcess) subProcess(task *Ttype) {
	te := p.workWithTask(task)
	go p.taskSorter(te)
}

// Analyze and fill reports variables
func (p *TProcess) fillReports() {
	go func() {
		for r := range p.undoneChan {
			p.errors = append(p.errors, r)
		}
		close(p.undoneChan)
	}()

	go func() {
		for r := range p.doneChan {
			p.results[r.id] = r
		}
		close(p.doneChan)
	}()
}

// Return filled reports
func (p *TProcess) getReports() (map[int64]*Ttype, []error) {
	return p.results, p.errors
}

// Start all async for:
// - start generating tasks
// - process separately tasks
// - write result to report
func (p *TProcess) processLoop() {
	go p.taskCreturer()
	go p.processTasks()
	go p.fillReports()
}

// Send signal to stop generating tasks
func (p *TProcess) stopTaskCreature() {
	p.finish <- struct{}{}
}

func main() {
	process := createProcess()
	process.processLoop()

	time.Sleep(WaitingTime)
	process.stopTaskCreature()

	result, err := process.getReports()

	println("Errors:")
	for r := range err {
		println(r)
	}

	println("Done tasks:")
	for r := range result {
		println(r)
	}
}
