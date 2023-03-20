package config

import (
	log "github.com/DggHQ/dggarchiver-logger"
	docker "github.com/docker/docker/client"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type DockerConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Network      string `yaml:"network"`
	DockerSocket *docker.Client
}

type K8sConfig struct {
	Enabled           bool   `yaml:"enabled"`
	Namespace         string `yaml:"namespace"`
	CPULimitConfig    string `yaml:"cpu_limit"`
	MemoryLimitConfig string `yaml:"memory_limit"`
	K8sClientSet      *kubernetes.Clientset
	CPUQuantity       resource.Quantity
	MemoryQuantity    resource.Quantity
}

type Controller struct {
	Verbose bool
	Docker  DockerConfig `yaml:"docker"`
	K8s     K8sConfig    `yaml:"k8s"`
	Plugins PluginConfig `yaml:"plugins"`
}

func (controller *Controller) loadDocker() {
	var err error

	controller.Docker.DockerSocket, err = docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		log.Fatalf("Wasn't able to connect to the docker socket: %s", err)
	}
}

func (controller *Controller) loadK8sConfig() {
	clusterConfig, err := rest.InClusterConfig()

	if cpuLimit, err := resource.ParseQuantity(controller.K8s.CPULimitConfig); err != nil {
		log.Fatalf("Could not parse K8S_CPU_LIMIT: %s", err)
	} else {
		controller.K8s.CPUQuantity = cpuLimit
	}
	if memoryLimit, err := resource.ParseQuantity(controller.K8s.MemoryLimitConfig); err != nil {
		log.Fatalf("Could not parse K8S_MEMORY_LIMIT: %s", err)
	} else {
		controller.K8s.MemoryQuantity = memoryLimit
	}

	if err != nil {
		log.Fatalf("Could not get k8s cluster config: %s", err)
	}
	clientSet, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		log.Fatalf("Could not create new client set config: %s", err)
	}
	controller.K8s.K8sClientSet = clientSet
}

func (controller *Controller) initialize() {
	// Docker and K8s
	if controller.Docker.Enabled && controller.K8s.Enabled {
		log.Fatalf("Please only enable one container orchestration backend")
	}

	switch true {
	case controller.Docker.Enabled:
		if controller.Docker.Network == "" {
			log.Fatalf("Please set the controller:docker:network config variable and restart the service")
		}
		controller.loadDocker()
	case controller.K8s.Enabled:
		if controller.K8s.Namespace == "" {
			log.Fatalf("Please set controller:k8s:namespace config variable when using K8s as a container orcherstration backend")
		}
		if controller.K8s.CPULimitConfig == "" {
			log.Fatalf("Please set controller:k8s:cpu_limit config variable when using K8s as a container orcherstration backend")
		}
		if controller.K8s.MemoryLimitConfig == "" {
			log.Fatalf("Please set controller:k8s:memory_limit config variable when using K8s as a container orcherstration backend")
		}
		controller.loadK8sConfig()
	}

	// Lua Plugins
	if controller.Plugins.Enabled {
		if controller.Plugins.PathToPlugin == "" {
			log.Fatalf("Please set the controller:plugins:path config variable and restart the service")
		}
	}
}
