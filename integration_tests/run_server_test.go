package servertest

import (
	"context"
	"encoding/json"
	"fmt"
	"goflow/config"
	"goflow/dags"
	"goflow/k8sclient"
	"goflow/logs"
	"goflow/orchestrator"
	"goflow/testutils"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	batch "k8s.io/api/batch/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var configPath string
var dagCount int

func adjustConfigDagPath(configPath string, dagPath string) string {
	fixedConfig := &config.GoFlowConfig{}
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(configBytes, fixedConfig)
	fixedConfig.DAGPath = dagPath
	fmt.Println(fixedConfig)
	newConfigPath := filepath.Join(testutils.GetTestFolder(), "tmp_config.json")
	fixedConfig.SaveConfig(newConfigPath)
	return newConfigPath
}

func getJobs(kubeClient kubernetes.Interface) *[]*batch.Job {
	jobSlice := make([]*batch.Job, 0)
	namespaces, err := kubeClient.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, namespace := range namespaces.Items {
		jobList, err := kubeClient.BatchV1().Jobs(
			namespace.Name,
		).List(
			context.TODO(),
			v1.ListOptions{},
		)
		if err != nil {
			panic(err)
		}
		for _, job := range jobList.Items {
			jobSlice = append(jobSlice, &job)
		}
	}
	return &jobSlice
}

func createDirIfNotExist(directory string) string {
	_, err := os.Stat(directory)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(directory, 0755)
		if errDir != nil {
			panic(err)
		}
	}
	return directory
}

func testStart(
	orchestrator orchestrator.Orchestrator,
	cycleDuration time.Duration,
	kubeClient kubernetes.Interface,
	breakLoop *bool,
) {
	dagCodes := []string{}
	for !*breakLoop {
		orchestrator.CollectDAGs()
		time.Sleep(cycleDuration)
		for _, dag := range orchestrator.DAGs() {
			dagCodes = append(dagCodes, dag.Code)
		}
		// fmt.Println("Jobs collected so far ", dagCodes)
		// fmt.Println("I'm here")
	}
}

func createFakeDagFile(dagFolder string, dagNum int) {
	fakeDagName := fmt.Sprintf("dag_file_%d.json", dagNum)
	filePath := filepath.Join(dagFolder, fakeDagName)
	fakeDagConfig := &dags.DAGConfig{Name: fakeDagName,
		Namespace:   "default",
		Schedule:    "* * * * *",
		Command:     fmt.Sprintf("echo %d", dagNum),
		Parallelism: 0,
		TimeLimit:   0,
		Retries:     2}
	jsonContent := fakeDagConfig.Marshal()
	ioutil.WriteFile(filePath, jsonContent, 0755)
	fmt.Println(fakeDagConfig.JSON())
}

// createFakeDags creates fake dag files, and returns their location
func createFakeDags(testFolder string) string {
	dagDir := createDirIfNotExist(filepath.Join(testFolder, "tmp_dags"))
	for i := 0; i < dagCount; i++ {
		createFakeDagFile(dagDir, i)
	}
	return dagDir
}

func TestMain(m *testing.M) {
	dagCount = 100
	fakeDagsPath := createFakeDags(testutils.GetTestFolder())
	defer os.RemoveAll(fakeDagsPath)
	configPath = adjustConfigDagPath(testutils.GetConfigPath(), fakeDagsPath)
	defer os.Remove(configPath)
	m.Run()
}

func TestStartServer(t *testing.T) {
	kubeClient := k8sclient.CreateKubeClient()
	defer testutils.CleanUpJobs(kubeClient)
	orch := *orchestrator.NewOrchestrator(configPath)
	loopBreaker := false
	go testStart(orch, 3, kubeClient, &loopBreaker)

	time.Sleep(2 * time.Second)
	loopBreaker = true

	logs.InfoLogger.Println("Dags length", len(orch.DAGs()))

	if len(orch.DAGs()) != dagCount {
		t.Errorf("DAG list does not have expected length")
	}
}
