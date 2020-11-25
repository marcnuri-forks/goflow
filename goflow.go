package main

import (
	"flag"
	"goflow/k8sclient"
	"goflow/logs"
	"goflow/orchestrator"
	"goflow/podutils"
	"time"
)

func main() {
	configPath := flag.String(
		"path",
		podutils.GetConfigPath(),
		"The path to the configuration file",
	)
	flag.Parse()

	kubeClient := k8sclient.CreateKubeClient()

	defer podutils.CleanUpPods(kubeClient)
	orch := *orchestrator.NewOrchestrator(*configPath)
	loopBreaker := make(chan struct{}, 1)
	go orch.Start(1, loopBreaker)

	time.Sleep(4 * time.Second)
	loopBreaker <- struct{}{}

	logs.InfoLogger.Println("Dags length", len(orch.DAGs()))
}
