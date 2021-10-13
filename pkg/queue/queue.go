package queue

import (
	"sync"
	"time"

	"github.com/coryschwartz/gateway-monitor/pkg/task"
)

type TaskQueue struct {
	mu    sync.Mutex
	tasks []task.Task
}

func (q *TaskQueue) Len() int {
	return len(q.tasks)
}

// TODO: Skop adding tasks that are already queued
// and maybe add a monitor for queue length, or queue fails
func (q *TaskQueue) Push(t ...task.Task) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tasks = append(q.tasks, t...)
}

func (q *TaskQueue) Pop() (task.Task, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.Len() == 0 {
		return nil, false
	}
	t := q.tasks[0]
	q.tasks = q.tasks[1:]
	return t, true
}

func (q *TaskQueue) Subscribe() chan task.Task {
	ch := make(chan task.Task)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				t, ok := q.Pop()
				if ok {
					ch <- t
				}
			}
		}
	}()
	return ch
}
