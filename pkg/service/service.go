package service

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// IService definition
type IService interface {
	Initialize(confFile string) error
	Stop()
	Run() error
}

const (
	ServiceRunning = "running"
	ServiceFailed  = "failed"
	ServiceMissing = "missing"
	ServiceStop    = "stopped"
)

//Service description
type Service struct {
	Name        string        `json:"name"`
	Systemd     []string      `json:"systemd"` //systemd service
	Version     string        `json:"version"`
	PackageName string        `json:"packageName"` //DebianPackageName
	Config      ServiceConfig `json:"config"`
	ConfigPath  string        `json:"configPath"`
}

//ServiceConfig desription
type ServiceConfig struct {
	LocalBroker   Broker      `json:"localBroker"`
	NetworkBroker Broker      `json:"networkBroker"`
	DB            DBConnector `json:"db"`
	LogLevel      string      `json:"logLevel"`
}

//DBConnector description
type DBConnector struct {
	ClientIP   string  `json:"clientIp"`
	ClientPort string  `json:"clientPort"`
	DBCluster  Cluster `json:"dbCluster"`
}

//Broker description
type Broker struct {
	IP       string `json:"ip"`
	Port     string `json:"port"`
	Login    string `json:"login"`
	Password string `json:"password"`
	CaPath   string `json:"caPath"`
	KeyPath  string `json:"keyPath"`
}

//Cluster description
type Cluster struct {
	Connectors []Connector `json:"connectors"`
}

//Connector description
type Connector struct {
	IP   string `json:"ip"`
	Port string `json:"port"`
}

//ServiceStatus description
type ServiceStatus struct {
	Service
	Status *string `json:"status"` //enable/running/disable etc.
}

//ReadServiceConfig parse the configuration file
func ReadServiceConfig(path string) (*ServiceConfig, error) {
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var config ServiceConfig

	json.Unmarshal(byteValue, &config)
	if config.LogLevel == "" {
		config.LogLevel = "INFO"
	}
	return &config, nil
}

//WriteServiceConfig store configuration
func WriteServiceConfig(path string, config ServiceConfig) error {
	dump, err := config.ToJSON()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, []byte(dump), 0644)
}

// ToJSON dump switch config struct
func (config ServiceConfig) ToJSON() (string, error) {
	inrec, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(inrec[:]), err
}

//ToService convert map interface to Service object
func ToService(val interface{}) (*Service, error) {
	var service Service
	inrec, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(inrec, &service)
	return &service, err
}

// GetServiceStatus return service status
func (s Service) GetServiceStatus() string {
	outputActive := &bytes.Buffer{}
	cmd := exec.Command("systemctl", "is-active", s.Name)
	cmd.Stdout = outputActive
	cmd.Run()
	output := strings.TrimSpace(string(outputActive.Bytes()))
	switch output {
	case "failed":
		return ServiceFailed
	case "active":
		return ServiceRunning
	default:
		outputEnable := &bytes.Buffer{}
		cmd = exec.Command("systemctl", "is-enabled", s.Name)
		cmd.Stdout = outputEnable
		cmd.Run()
		output = strings.TrimSpace(string(outputEnable.Bytes()))
		if output == "disabled" {
			return ServiceStop
		}
		return ServiceMissing
	}
}

// Install install a given service
func (s Service) Install() (string, error) {
	cmd := exec.Command("apt-get", "install", "-y", s.PackageName)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Remove a given service
func (s Service) Remove() (string, error) {
	cmd := exec.Command("apt-get", "remove", "-y", s.PackageName)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Start a given service
func (s Service) Start() (string, error) {
	cmd := exec.Command("systemctl", "start", s.Name)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Stop a given service
func (s Service) Stop() (string, error) {
	cmd := exec.Command("systemctl", "stop", s.Name)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

//InstallPackages start all given services
func InstallPackages(services map[string]Service) {
	for _, service := range services {
		service.Install()
	}
}

//StartServices start all given services
func StartServices(services map[string]Service) {
	for _, service := range services {
		status := service.GetServiceStatus()
		if status != "active" {
			service.Start()
		}
	}
}

//RemoveServices remove all given services
func RemoveServices(services map[string]Service) {
	for _, service := range services {
		service.Stop()
		service.Remove()
	}
}

//GetPackageVersion return package version
func GetPackageVersion(service string) *string {
	cmd := exec.Command("apt", "show", service)
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput
	err := cmd.Run()
	if err != nil {
		return nil
	}
	output := string(cmdOutput.Bytes())
	for _, line := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
		if !strings.HasPrefix(line, "Version:") {
			continue
		}
		lineSplit := strings.Split(line, " ")
		if len(lineSplit) > 1 {
			version := lineSplit[1]
			return &version
		}
	}

	return nil
}
