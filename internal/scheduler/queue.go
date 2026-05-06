package scheduler

import "container/heap"

// workItem is one entry waiting in the priority queue.
type workItem struct {
	job    *Job
	seq    int64 // monotonically increasing FIFO tiebreaker
	index  int   // heap index (maintained by heap.Interface)
}

// priorityQueue is a min-heap ordered by descending priority then ascending seq.
// (highest priority value = dequeued first)
type priorityQueue []*workItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].job.Priority != pq[j].job.Priority {
		return pq[i].job.Priority > pq[j].job.Priority // higher priority first
	}
	return pq[i].seq < pq[j].seq // FIFO within same priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x any) {
	item := x.(*workItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[:n-1]
	return item
}

// jobQueue wraps the heap with a mutex-free interface (callers hold the
// Scheduler's mu lock when interacting with it).
type jobQueue struct {
	pq  priorityQueue
	seq int64
}

func newJobQueue() *jobQueue { return &jobQueue{} }

func (q *jobQueue) Push(j *Job) {
	q.seq++
	item := &workItem{job: j, seq: q.seq}
	heap.Push(&q.pq, item)
}

func (q *jobQueue) Pop() *Job {
	if q.pq.Len() == 0 {
		return nil
	}
	return heap.Pop(&q.pq).(*workItem).job
}

func (q *jobQueue) Len() int { return q.pq.Len() }
