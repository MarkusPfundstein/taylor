package server

import (
	"time"
	"fmt"
	"os"
	"sync"
	"sort"

	"taylor/server/database"
	"taylor/lib/structs"
	"taylor/lib/util"
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
		ok := util.IsSubsetString(restrict, node.Capabilities)
		if ok == true {
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

func nodesWhichFulfillGpuRequirements(nodes []*Node, gpuReqs []structs.GpuRequirement) []*Node {
	outNodes := make([]*Node, 0)

	proxyNodes := make([]*Node, 0)

	// deep copy of each node. we dont want to modify original data
	for _, node := range nodes { 
		// filter out all nodes that have less gpus than requested
		if len(node.GpuInfo) >= len(gpuReqs) {
			proxyNodes = append(proxyNodes, node)
		}
	}

	if len(proxyNodes) == 0 {
		return outNodes
	}

	for _, n := range proxyNodes{
		// for each gpu requirement, go through nodes info and see if we find match
		foundN := 0
		usedInfos := make(map[int]int, len(n.GpuInfo))
		for _, gpuReq := range gpuReqs {
			foundMatch := false
			for k, gpuInfo := range n.GpuInfo {
				if usedInfos[k] == 1 {
					continue
				}
				if gpuReq.Type != "" && gpuInfo.NameGPU != gpuReq.Type {
					continue
				}
				if gpuReq.MemoryAvailable > -1 && gpuInfo.MemoryFreeMB < gpuReq.MemoryAvailable {
					continue
				}
				foundMatch = true
				// add memory so that we dont reuse it next iteration
				n.GpuInfo[k].MemoryFreeMB -= gpuReq.MemoryAvailable
				usedInfos[k] = 1
				break
			}
			// we havent found a match for our gpu
			if foundMatch {
				foundN++
			}
			if foundN == len(gpuReqs) {
				break
			}
		}
		if foundN == len(gpuReqs) {
			outNodes = append(outNodes, n)
		}
	}

	return outNodes
}

func printNodes(nodes []*Node) {
	for _, n := range nodes {
		fmt.Printf("%+v\n", *n)
	}
}

func distribute(nodesIn []*Node, jobs []*structs.Job) []NodeJobMap{

	proxyNodes := make([]*Node, len(nodesIn ))

	// deep copy of each node. we dont want to modify original data
	for idx, node := range nodesIn {
		copy := &Node{
			Name:	      node.Name,
			Capabilities: node.Capabilities,
			Capacity:     node.Capacity,
			JobsRunning:  node.JobsRunning,
			GpuInfo:      make([]structs.GpuInfo, len(node.GpuInfo)),
		}
		for k, gpuInfo := range node.GpuInfo {
			copy.GpuInfo[k] = structs.GpuInfo{
				NameGPU:	gpuInfo.NameGPU,
				Temperature:	gpuInfo.Temperature,
				MemoryTotalMB:	gpuInfo.MemoryTotalMB,
				MemoryFreeMB:	gpuInfo.MemoryFreeMB,
				Utilization:	gpuInfo.Utilization,
			}
		}
		proxyNodes[idx] = copy
	}

	nodeProxyJobMaps := make([]NodeJobMap, 0)

	for _, job := range jobs {
		// find all nodes that have some space left
		freeNodes := freeNodes(proxyNodes)
		if len(freeNodes) == 0 {
			break
		}

		// filter out all nodes that are not capable of handling the job
		capableNodes := nodesWithCapabilities(job.Restrict, freeNodes )
		if len(capableNodes) == 0 {
			continue
		}

		// sort by capability. the ones with the least comes first
		sortCapableNodes(capableNodes)

		if len(job.GpuRequirement) > 0 {
			gpuNodes := nodesWhichFulfillGpuRequirements(capableNodes, job.GpuRequirement)
			if len(gpuNodes) == 0 {
				continue
			}

			capableNodes = gpuNodes
		}

		// we have now multiple capable nodes. take first one
		nodeProxyJobMaps = append(nodeProxyJobMaps, NodeJobMap{
			node: capableNodes[0],
			job: job,
		})
		capableNodes[0].JobsRunning++
	}

	return nodeProxyJobMaps
}

func sortJobsByPriority(jobs []*structs.Job) {
	sort.Slice(jobs[:], func (i int, j int) bool {
		return jobs[i].Priority > jobs[j].Priority
	})
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

		sortJobsByPriority(jobs)

		distributed := distribute(s.tcpServer.Nodes(), jobs)

		// schedule all jobs on corresponding nodes
		var wg sync.WaitGroup
		for _, v := range distributed {
			fmt.Printf("Schedule job %s [%v] to node %s [%v]\n", v.job.Identifier, v.job.Restrict, v.node.Name, v.node.Capabilities)

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
