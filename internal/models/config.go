package models

type Service struct{
	Name string `yaml:"name"`
	URL string `yaml:"url"`
}

type Config struct {
	Services []Service `yaml:"services"`
}
