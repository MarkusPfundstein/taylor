package server

import (
	"time"
	"fmt"
	"os"
	"sync"
	"sort"

	"taylor/server/database"
	"taylor/lib/structs"
	"taylor/lib/tcp"
)

type Scheduler struct {
	sleepMs		time.Duration
	tcpServer	*TcpServer
	store		*database.Store
	config		Config
}

type NodeJobMap struct {
	node *Node
	job  *structs.Job 
}

func removeNode(slice []*Node, s int) []*Node{
    return append(slice[:s], slice[s+1:]...)
}

func nodesWithCapabilities(restrict []string, nodes []*Node) []*Node{
	res := []*Node{}
	
	for _, node := range nodes {
		nope := false
		for _, requirement := range restrict {
			isInIt := false
			for _, capability := range node.Capabilities {
				if requirement == capability {
					isInIt = true
					break
				}
			}
			if isInIt == false {
				nope = true
			}
		}
		if nope == false {
			res = append(res, node)
		}
	}
	
	return res
}

func sortCapableNodes(nodes []*Node) {
	sort.Slice(nodes[:], func (i int, j int) bool {
		return len(nodes[i].Capabilities) < len(nodes[j].Capabilities)
	})
}

func freeNodes(nodes []*Node) []*Node {

	freeNodes := make([]*Node, 0)
	for _, node := range nodes {
		if node.Capacity > node.JobsRunning {
			freeNodes = append(freeNodes, node)
		}
	}
	return freeNodes
}

func distribute(nodesIn []*Node, jobs []*structs.Job) []NodeJobMap{

	proxyNodes := make([]*Node, len(nodesIn ))

	for idx, node := range nodesIn {
		copy := *node
		proxyNodes[idx] = &copy
	}

	nodeProxyJobMaps := make([]NodeJobMap, 0)
	
	for _, job := range jobs {
		// find all nodes with right capabilities
		freeNodes := freeNodes(proxyNodes)
		
		// get all nodes that are capable of handling the job
		capableNodes := nodesWithCapabilities(job.Restrict, freeNodes )
		
		// sort by capability. the ones with the least take it first
		sortCapableNodes(capableNodes)
	
		for idx, node := range capableNodes{
			if (node.Capacity - node.JobsRunning) > 0 {
				nodeProxyJobMaps = append(nodeProxyJobMaps, NodeJobMap{
					node: node,
					job: job,
				})
				node.JobsRunning++
				// remove preliminarily
				if (node.Capacity - node.JobsRunning) > 0 {
					removeNode(capableNodes, idx)
				}
				break
			} else {			
				removeNode(capableNodes, idx)
			}
		}
	}
	
	return nodeProxyJobMaps
}

func (s *Scheduler) schedule () {
	for {
		time.Sleep(s.sleepMs * time.Millisecond)
		
		jobs, err := s.store.JobsWithStatus(structs.JOB_STATUS_WAITING, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error %v\n", err)
			// what shall we do?
			panic(err)
		}
		if len(jobs) == 0 {
			continue
		}

		fmt.Printf("distribute %d over %d nodes\n", len(jobs), len(s.tcpServer.Nodes()))

		distributed := distribute(s.tcpServer.Nodes(), jobs)

		// try to schedule all jobs on corresponding nodes
		var wg sync.WaitGroup
		for _, v := range distributed {
			fmt.Printf("%s [%v]-> %s [%v]\n", v.job.Identifier, v.job.Restrict, v.node.Name, v.node.Capabilities)

			wg.Add(1)
			go func (njm NodeJobMap) {
				defer wg.Done()

				payload := &tcp.MsgNewJobOffer{
					MsgBase: tcp.MsgBase{
						Command: tcp.MSG_NEW_JOB_OFFER,
						NodeName: s.config.Name,
					},
					Job: *njm.job,
				}

				s.tcpServer.Unicast(njm.node, payload)
			}(v)
		}

		wg.Wait()
	}
}

func StartScheduler(config Config, store *database.Store, server *TcpServer) {
	scheduler := Scheduler{
		sleepMs: 10000,
		store: store,
		tcpServer: server,
		config: config,
	}

	go scheduler.schedule()
}
