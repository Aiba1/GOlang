package main

import (
	"fmt"
	"log"

	"textanalyzer-cp1/analyzer"
	"textanalyzer-cp1/pool"
	"textanalyzer-cp1/service"
	"textanalyzer-cp1/task"
)

const (
	workers  = 3
	queueCap = 10
)

func main() {
	store := task.NewStore()
	svc := service.NewTaskService(store, analyzer.New())

	fmt.Println("== CP1: synchronous processing still works ==")
	t0, err := svc.CreateTask("Hello Go world")
	if err != nil {
		log.Fatal(err)
	}
	t0, err = svc.ProcessTask(t0.ID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s %s %s\n", t0.ID, t0.Status, t0.Result.Summary())
	if _, err := svc.CreateTask("   "); err != nil {
		fmt.Println("empty input rejected:", err)
	}

	fmt.Printf("\n== CP2: worker pool (%d workers, queue capacity %d) ==\n", workers, queueCap)
	p, err := pool.New(store, analyzer.New(), workers, queueCap)
	if err != nil {
		log.Fatal(err)
	}

	inputs := []string{
		"Go makes concurrency approachable",
		"Channels connect goroutines",
		"Do not communicate by sharing memory",
		"Share memory by communicating",
		"A worker pool bounds concurrency",
		"Mutexes protect shared state",
	}
	var ids []string
	for _, in := range inputs {
		t, err := svc.CreateTask(in)
		if err != nil {
			log.Fatal(err)
		}
		if err := p.Submit(t.ID); err != nil {
			log.Fatal(err)
		}
		ids = append(ids, t.ID)
		fmt.Printf("submitted %s (Submit returned without waiting)\n", t.ID)
	}

	if err := p.Submit(ids[0]); err != nil {
		fmt.Println("duplicate submit rejected:", err)
	}
	if err := p.Submit(t0.ID); err != nil {
		fmt.Println("completed task submit rejected:", err)
	}

	p.Shutdown() // stop accepting, drain the queue, wait for workers
	fmt.Println("\n== after shutdown ==")
	for _, id := range ids {
		t, err := svc.GetTask(id)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s %s %s\n", t.ID, t.Status, t.Result.Summary())
	}

	late, err := svc.CreateTask("too late")
	if err != nil {
		log.Fatal(err)
	}
	if err := p.Submit(late.ID); err != nil {
		fmt.Println("submit after shutdown rejected:", err)
	}
}
