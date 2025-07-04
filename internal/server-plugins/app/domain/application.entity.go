package app

import (
	"fmt"
	"time"

	"github.com/alex-galey/dokku-mcp/internal/shared"
	"github.com/alex-galey/dokku-mcp/internal/shared/process"
)

type Application struct {
	name *ApplicationName

	state     *ApplicationState
	createdAt time.Time
	updatedAt time.Time

	configuration *ApplicationConfiguration

	deploymentInfo *DeploymentInfo

	events []DomainEvent
}

type ApplicationConfiguration struct {
	buildpack       *shared.BuildpackName
	domains         []*shared.DomainName
	environmentVars map[shared.EnvVarKey]*shared.EnvVarValue
	processes       map[process.ProcessType]*process.Process
}

type DeploymentInfo struct {
	currentGitRef   *shared.GitRef
	lastDeployedAt  *time.Time
	buildImage      *shared.DockerImage
	runImage        *shared.DockerImage
	deploymentCount int
}

type DomainEvent interface {
	OccurredAt() time.Time
	EventType() string
	AggregateID() string
}

func NewApplication(name string) (*Application, error) {
	// Default to "exists" state for new applications
	return NewApplicationWithState(name, StateExists)
}

// NewApplicationWithState creates an application with a specific state
// This is useful for repositories to create entities that reflect actual system state
func NewApplicationWithState(name string, state StateValue) (*Application, error) {
	appName, err := NewApplicationName(name)
	if err != nil {
		return nil, fmt.Errorf("unable to create application: %w", err)
	}

	applicationState, err := NewApplicationState(state)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize state: %w", err)
	}

	app := &Application{
		name:      appName,
		state:     applicationState,
		createdAt: time.Now(),
		updatedAt: time.Now(),
		configuration: &ApplicationConfiguration{
			domains:         make([]*shared.DomainName, 0),
			environmentVars: make(map[shared.EnvVarKey]*shared.EnvVarValue),
			processes:       make(map[process.ProcessType]*process.Process),
		},
		deploymentInfo: &DeploymentInfo{
			deploymentCount: 0,
		},
		events: make([]DomainEvent, 0),
	}

	// Publish creation event
	app.addEvent(NewApplicationCreatedEvent(appName.Value(), time.Now()))

	return app, nil
}

func (a *Application) Name() *ApplicationName { return a.name }

func (a *Application) State() *ApplicationState { return a.state }

func (a *Application) CreatedAt() time.Time { return a.createdAt }

func (a *Application) UpdatedAt() time.Time { return a.updatedAt }

func (a *Application) Configuration() *ApplicationConfiguration {
	return a.copyConfiguration()
}

func (a *Application) Deploy(gitRef *shared.GitRef, buildOpts *DeploymentOptions) error {
	if gitRef == nil {
		return fmt.Errorf("git reference cannot be null")
	}

	a.deploymentInfo.currentGitRef = gitRef
	now := time.Now()
	a.deploymentInfo.lastDeployedAt = &now
	a.deploymentInfo.deploymentCount++

	if buildOpts != nil {
		a.deploymentInfo.buildImage = buildOpts.BuildImage
		a.deploymentInfo.runImage = buildOpts.RunImage
	}

	a.updatedAt = time.Now()
	a.addEvent(NewApplicationDeployedEvent(a.name.Value(), gitRef.Value(), time.Now()))

	return nil
}

// CompleteDeployment just sets state to running
func (a *Application) CompleteDeployment() error {
	return a.setState(StateRunning)
}

// FailDeployment sets state to error
func (a *Application) FailDeployment(reason string) error {
	a.addEvent(NewApplicationDeploymentFailedEvent(a.name.Value(), reason, time.Now()))
	return a.setState(StateError)
}

func (a *Application) Scale(processType process.ProcessType, instances int) error {
	proc, exists := a.configuration.processes[processType]
	if !exists {
		// Create a process for scaling (command will be determined from Procfile later)
		proc, err := process.NewProcessForScaling(processType, instances)
		if err != nil {
			return fmt.Errorf("unable to create process: %w", err)
		}
		a.configuration.processes[processType] = proc
		a.updatedAt = time.Now()
		a.addEvent(NewApplicationScaledEvent(a.name.Value(), string(processType), 0, instances, time.Now()))
		return nil
	}

	oldScale := proc.Scale()
	err := proc.SetScale(instances)
	if err != nil {
		return err
	}
	a.updatedAt = time.Now()
	a.addEvent(NewApplicationScaledEvent(a.name.Value(), string(processType), oldScale, instances, time.Now()))

	return nil
}

func (a *Application) AddDomain(domainName string) error {
	domainVO, err := shared.NewDomainName(domainName)
	if err != nil {
		return fmt.Errorf("invalid domain: %w", err)
	}

	for _, existingDomain := range a.configuration.domains {
		if existingDomain.Equal(domainVO) {
			return fmt.Errorf("the domain %s already exists", domainName)
		}
	}

	a.configuration.domains = append(a.configuration.domains, domainVO)
	a.updatedAt = time.Now()
	a.addEvent(NewDomainAddedEvent(a.name.Value(), domainName, time.Now()))

	return nil
}

func (a *Application) RemoveDomain(domainName string) error {
	domainVO, err := shared.NewDomainName(domainName)
	if err != nil {
		return fmt.Errorf("invalid domain: %w", err)
	}

	for i, existingDomain := range a.configuration.domains {
		if existingDomain.Equal(domainVO) {
			// Delete the domain
			a.configuration.domains = append(a.configuration.domains[:i], a.configuration.domains[i+1:]...)
			a.updatedAt = time.Now()
			a.addEvent(NewDomainRemovedEvent(a.name.Value(), domainName, time.Now()))
			return nil
		}
	}

	return fmt.Errorf("the domain %s doesn't exist", domainName)
}

func (a *Application) SetBuildpack(buildpackName string) error {
	buildpackVO, err := shared.NewBuildpackName(buildpackName)
	if err != nil {
		return fmt.Errorf("invalid buildpack: %w", err)
	}

	a.configuration.buildpack = buildpackVO
	a.updatedAt = time.Now()
	a.addEvent(NewBuildpackChangedEvent(a.name.Value(), buildpackName, time.Now()))

	return nil
}

func (a *Application) SetEnvironmentVariable(key, value string) error {
	envKey, err := shared.NewEnvVarKey(key)
	if err != nil {
		return err
	}
	envValue := shared.NewEnvVarValue(value)

	a.configuration.environmentVars[*envKey] = envValue
	a.updatedAt = time.Now()

	return nil
}

func (a *Application) AddProcess(processType process.ProcessType, command string, scale int) error {
	proc, err := process.NewProcess(processType, command, scale)
	if err != nil {
		return fmt.Errorf("unable to add process: %w", err)
	}

	a.configuration.processes[processType] = proc
	a.updatedAt = time.Now()

	return nil
}

// AddProcessForScaling adds a process for scaling without requiring a command.
// This is used when the command will be determined from the Procfile later.
func (a *Application) AddProcessForScaling(processType process.ProcessType, scale int) error {
	proc, err := process.NewProcessForScaling(processType, scale)
	if err != nil {
		return fmt.Errorf("unable to add process for scaling: %w", err)
	}

	a.configuration.processes[processType] = proc
	a.updatedAt = time.Now()

	return nil
}

func (a *Application) IsRunning() bool {
	return a.state.Value() == StateRunning
}

func (a *Application) IsDeployed() bool {
	return a.state.IsDeployed()
}

func (a *Application) HasDomain(domainName string) bool {
	domainVO, err := shared.NewDomainName(domainName)
	if err != nil {
		return false
	}

	for _, existingDomain := range a.configuration.domains {
		if existingDomain.Equal(domainVO) {
			return true
		}
	}
	return false
}

func (a *Application) GetProcessScale(processType process.ProcessType) int {
	if proc, exists := a.configuration.processes[processType]; exists {
		return proc.Scale()
	}
	return 0
}

func (a *Application) GetDomains() []string {
	domains := make([]string, len(a.configuration.domains))
	for i, domainVO := range a.configuration.domains {
		domains[i] = domainVO.Value()
	}
	return domains
}

func (a *Application) GetEvents() []DomainEvent {
	return a.events
}

func (a *Application) ClearEvents() {
	a.events = make([]DomainEvent, 0)
}

// Private methods

// setState replaces the complex changeState logic with simple state setting
func (a *Application) setState(newState StateValue) error {
	newStateObj, err := NewApplicationState(newState)
	if err != nil {
		return fmt.Errorf("invalid state transition to %s: %w", newState, err)
	}

	a.state = newStateObj
	a.updatedAt = time.Now()
	return nil
}

func (a *Application) addEvent(event DomainEvent) {
	a.events = append(a.events, event)
}

func (a *Application) copyConfiguration() *ApplicationConfiguration {
	domains := make([]*shared.DomainName, len(a.configuration.domains))
	copy(domains, a.configuration.domains)

	envVars := make(map[shared.EnvVarKey]*shared.EnvVarValue)
	for k, v := range a.configuration.environmentVars {
		envVars[k] = v
	}

	processes := make(map[process.ProcessType]*process.Process)
	for k, v := range a.configuration.processes {
		processes[k] = v // This is a shallow copy, but Process is now an entity-like object
	}

	return &ApplicationConfiguration{
		buildpack:       a.configuration.buildpack,
		domains:         domains,
		environmentVars: envVars,
		processes:       processes,
	}
}

type DeploymentOptions struct {
	BuildImage *shared.DockerImage
	RunImage   *shared.DockerImage
	ForceClean bool
	NoCache    bool
}

// ApplicationInfo represents application info for JSON serialization
type ApplicationInfo struct {
	Name       string    `json:"name"`
	State      string    `json:"state"`
	IsRunning  bool      `json:"is_running"`
	IsDeployed bool      `json:"is_deployed"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ApplicationStatus represents detailed application status for JSON serialization
type ApplicationStatus struct {
	Name       string    `json:"name"`
	State      string    `json:"state"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	IsRunning  bool      `json:"is_running"`
	IsDeployed bool      `json:"is_deployed"`
	Domains    []string  `json:"domains"`
}

// ApplicationListData represents the application list resource data
type ApplicationListData struct {
	Applications []ApplicationInfo `json:"applications"`
	Count        int               `json:"count"`
}

// ApplicationSummaryData represents the application summary resource data
type ApplicationSummaryData struct {
	TotalApps    int `json:"total_apps"`
	RunningApps  int `json:"running_apps"`
	StoppedApps  int `json:"stopped_apps"`
	DeployedApps int `json:"deployed_apps"`
}
