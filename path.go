package main

import (
	"log"
	"os"
	"strings"
	"time"
)

const delimiter = "/"

type pathConfig struct {
}

type PathConfigurator interface {
	Configure(base, url string) string
}

func NewPathConfigurator() PathConfigurator {
	return &pathConfig{}
}

func (p *pathConfig) Configure(base, url string) string {
	backupPath := p.compose(base, url)

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		err = os.MkdirAll(backupPath, os.ModePerm)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}

	return backupPath
}

func (p *pathConfig) compose(base, url string) string {
	sb := strings.Builder{}
	sb.WriteString(base)
	if base != "" && !strings.HasSuffix(base, delimiter) {
		sb.WriteString(delimiter)
	}
	parts := strings.Split(url, delimiter)
	if len(parts) > 2 {
		sb.WriteString(parts[2])
		sb.WriteString(delimiter)
	}
	t := time.Now()
	sb.WriteString(t.Format("2006-01-02"))
	sb.WriteString(delimiter)
	return sb.String()
}
