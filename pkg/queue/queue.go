package queue

import (
	"sync"
	"time"

	"github.com/coryschwartz/gateway-monitor/pkg/task"
)

type TaskQueue struct {
	mu      sync.Mutex
	tasks   []task.Task
	taskmap map[task.Task]bool
}

func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks:   []task.Task{},
		taskmap: make(map[task.Task]bool),
	}
}

func (q *TaskQueue) Len() int {
	return len(q.tasks)
}

// TODO:
// add a monitor for queue length, or queue fails
func (q *TaskQueue) Push(tsks ...task.Task) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, newtsk := range tsks {
		if _, found := q.taskmap[newtsk]; found {
			continue
		}
		q.tasks = append(q.tasks, newtsk)
		q.taskmap[newtsk] = true
	}
}

func (q *TaskQueue) Pop() (task.Task, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.Len() == 0 {
		return nil, false
	}
	t := q.tasks[0]
	q.tasks = q.tasks[1:]
	delete(q.taskmap, t)
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
