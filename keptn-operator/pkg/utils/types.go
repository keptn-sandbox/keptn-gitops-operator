package utils

//GitRepositoryConfig defines the configuration which is used by git components
type GitRepositoryConfig struct {
	RemoteURI string
	User      string
	Token     string
	Branch    string
}
