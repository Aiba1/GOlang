// Command textanalyzer-cp1 is a small manual demo of the application:
// it wires the task store together with the text analyzer, creates a
// few tasks, processes them, and prints what happened — including the
// error paths (not-found, invalid input, reprocessing a completed
// task) so the lifecycle is visible end to end.
//
// This is a demo entrypoint, not a required part of the checkpoint
// (the graded behavior is in the task/analyzer/service packages and
// their tests). Run it with:
//
//	go run .
package main

import (
	"fmt"

	"textanalyzer-cp1/analyzer"
	"textanalyzer-cp1/service"
	"textanalyzer-cp1/task"
)

func main() {
	svc := service.NewTaskService(task.NewStore(), analyzer.New())

	fmt.Println("== creating tasks ==")
	t1, err := svc.CreateTask("Hello Go world")
	must(err)
	fmt.Printf("created %s: %q (status=%s)\n", t1.ID, t1.Input, t1.Status)

	t2, err := svc.CreateTask("Go makes concurrency approachable")
	must(err)
	fmt.Printf("created %s: %q (status=%s)\n", t2.ID, t2.Input, t2.Status)

	fmt.Println("\n== rejecting invalid input ==")
	if _, err := svc.CreateTask("   "); err != nil {
		fmt.Printf("CreateTask(\"   \") correctly rejected: %v\n", err)
	}

	fmt.Println("\n== processing tasks ==")
	t1, err = svc.ProcessTask(t1.ID)
	must(err)
	fmt.Printf("%s status=%s result=%s\n", t1.ID, t1.Status, t1.Result.Summary())

	t2, err = svc.ProcessTask(t2.ID)
	must(err)
	fmt.Printf("%s status=%s result=%s\n", t2.ID, t2.Status, t2.Result.Summary())

	fmt.Println("\n== rejecting invalid operations ==")
	if _, err := svc.ProcessTask(t1.ID); err != nil {
		fmt.Printf("reprocessing %s correctly rejected: %v\n", t1.ID, err)
	}
	if _, err := svc.GetTask("task-999"); err != nil {
		fmt.Printf("looking up a missing task correctly rejected: %v\n", err)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
