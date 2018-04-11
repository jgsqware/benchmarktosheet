// Package config provides ...
package config

type Report struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Config struct {
	Reports []Report `json:"reports"`
}
