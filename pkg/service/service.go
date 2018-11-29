package service

// IService definition
type IService interface {
	Initialize(confFile string) error
	Stop()
	Run() error
}
