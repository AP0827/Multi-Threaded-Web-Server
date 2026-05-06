package pool

import (
	"MTWS/core"
	"net"
	"sync"
)

type WorkerPool struct {
	jobs chan Job
	wg   sync.WaitGroup
}

type Job struct {
	Conn    net.Conn
	Router  *core.Router
	Options core.ConnectionOptions
}

func worker(id int, jobs <-chan Job, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		/*handle job worker connection*/
		core.HandleConnectionWithOptions(job.Conn, job.Router, job.Options)
	}
}

func StartWorkerPool(numWorkers int, jobs chan Job) *WorkerPool {
	p := &WorkerPool{jobs: jobs}
	for i := 0; i < numWorkers; i++ {
		p.wg.Add(1)
		go worker(i, jobs, &p.wg)
	}
	return p
}

func (p *WorkerPool) Stop() {
	if p == nil {
		return
	}
	close(p.jobs)
	p.wg.Wait()
}
