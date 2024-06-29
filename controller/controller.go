package controller

import (
	"log/slog"
	"os"

	"github.com/DggHQ/dggarchiver-config/misc"
	docker "github.com/docker/docker/client"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type DockerConfig struct {
	Enabled      bool   `yaml:"enabled"`
	AutoRemove   bool   `yaml:"autoremove"`
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
	Verbose       bool
	WorkerImage   string             `yaml:"worker_image"`
	Docker        DockerConfig       `yaml:"docker"`
	K8s           K8sConfig          `yaml:"k8s"`
	Notifications misc.Notifications `yaml:"notifications"`
}

type Config struct {
	*Controller `yaml:"controller"`
	NATS        misc.NATSConfig `yaml:"nats"`
}

func New() *Config {
	var (
		err error
		cfg = Config{}
		lvl slog.LevelVar
	)

	misc.SetupSlog(&lvl)

	_ = godotenv.Load()

	configFile := os.Getenv("CONFIG")
	if configFile == "" {
		configFile = "config.yaml"
	}
	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		slog.Error("unable to load config", slog.Any("err", err))
		os.Exit(1)
	}

	err = yaml.Unmarshal(configBytes, &cfg)
	if err != nil {
		slog.Error("unable to unmarshall config yaml", slog.Any("err", err))
		os.Exit(1)
	}

	if cfg.Controller.Verbose {
		lvl.Set(slog.LevelDebug)
	}

	cfg.Controller.initialize()

	// NATS Host Name or IP
	if cfg.NATS.Host == "" {
		slog.Error("config variable not set", slog.String("var", "nats:host"))
		os.Exit(1)
	}
	// NATS Topic Name
	if cfg.NATS.Topic == "" {
		slog.Error("config variable not set", slog.String("var", "nats:topic"))
		os.Exit(1)
	}
	cfg.NATS.Load()

	return &cfg
}

func (controller *Controller) loadDocker() {
	var err error

	controller.Docker.DockerSocket, err = docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		slog.Error("unable to connect to the docker socket", slog.Any("err", err))
		os.Exit(1)
	}
}

func (controller *Controller) loadK8sConfig() {
	cpuLimit, err := resource.ParseQuantity(controller.K8s.CPULimitConfig)
	if err != nil {
		slog.Error("unable to parse k8s cpu limit", slog.Any("err", err))
		os.Exit(1)
	}
	controller.K8s.CPUQuantity = cpuLimit

	memoryLimit, err := resource.ParseQuantity(controller.K8s.MemoryLimitConfig)
	if err != nil {
		slog.Error("unable to parse k8s memory limit", slog.Any("err", err))
		os.Exit(1)
	}
	controller.K8s.MemoryQuantity = memoryLimit

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		slog.Error("unable to get k8s cluster config", slog.Any("err", err))
		os.Exit(1)
	}

	clientSet, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		slog.Error("unable to create k8s client set", slog.Any("err", err))
		os.Exit(1)
	}
	controller.K8s.K8sClientSet = clientSet
}

func (controller *Controller) initialize() {
	// Docker and K8s
	if controller.WorkerImage == "" {
		slog.Error("config variable not set", slog.String("var", "controller:worker_image"))
		os.Exit(1)
	}

	if controller.Docker.Enabled && controller.K8s.Enabled {
		slog.Error("too many orchestration backends enabled")
		os.Exit(1)
	}

	switch {
	case controller.Docker.Enabled:
		if controller.Docker.Network == "" {
			slog.Error("config variable not set", slog.String("var", "controller:docker:network"))
			os.Exit(1)
		}
		controller.loadDocker()
	case controller.K8s.Enabled:
		if controller.K8s.Namespace == "" {
			slog.Error("config variable not set", slog.String("var", "controller:k8s:namespace"))
			os.Exit(1)
		}
		if controller.K8s.CPULimitConfig == "" {
			slog.Error("config variable not set", slog.String("var", "controller:k8s:cpu_limit"))
			os.Exit(1)
		}
		if controller.K8s.MemoryLimitConfig == "" {
			slog.Error("config variable not set", slog.String("var", "controller:k8s:memory_limit"))
			os.Exit(1)
		}
		controller.loadK8sConfig()
	}

	// Notifications
	if controller.Notifications.Enabled() {
		var err error
		controller.Notifications.Sender, err = shoutrrr.CreateSender(controller.Notifications.List...)
		if err != nil {
			slog.Error("unable to create notification sender", slog.Any("err", err))
			os.Exit(1)
		}
	}
}
