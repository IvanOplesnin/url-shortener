package config

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/IvanOplesnin/url-shortener/internal/logger"
)

const (
	AddressKEY  = "SERVER_ADDRESS"
	BaseURLKEY  = "BASE_URL"
	FilePathKEY = "FILE_STORAGE_PATH"
)

type Server struct {
	Port int
	Host string
}

func (s *Server) String() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func (s *Server) Set(flagValue string) error {
	parts := strings.Split(flagValue, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid address %q: must be in form host:port", flagValue)
	}

	host := strings.TrimSpace(parts[0])
	portStr := strings.TrimSpace(parts[1])

	if host == "" {
		return fmt.Errorf("invalid host %q: host cannot be empty", host)
	}
	if strings.ContainsAny(host, " \t\n\r") {
		return fmt.Errorf("invalid host %q: host must not contain whitespace", host)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port %q: %w", portStr, err)
	}
	if port <= 0 || port > 65535 {
		return fmt.Errorf("port out of range: %d (must be 1–65535)", port)
	}

	s.Host = host
	s.Port = port
	return nil
}

func (s *Server) UnmarshalText(t []byte) error {
	return s.Set(string(t))
}

type Logger struct {
	Level  string           `env:"LOG_LEVEL"`
	Format logger.Formatter `env:"LOG_FORMAT"`
}

type Config struct {
	Server   Server `env:"SERVER_ADDRESS"`
	BaseURL  string `env:"BASE_URL"`
	Logger   Logger
	FilePath string `env:"FILE_STORAGE_PATH"`
}

func (c *Config) String() string {
	server := fmt.Sprintf("Server=%s", &c.Server)
	baseURL := fmt.Sprintf("BaseURl=%s", c.BaseURL)
	logLevel := fmt.Sprintf("LogLevel=%s", c.Logger.Level)
	logFormat := fmt.Sprintf("LogFormat=%s", c.Logger.Format)
	filePath := fmt.Sprintf("filePath=%s", c.FilePath)
	return strings.Join([]string{server, baseURL, logLevel, logFormat, filePath}, "; ") + "\n"
}

func GetConfig() (*Config, error) {
	const (
		baseURLFlagUsage = `Base URL, e.g. "http://localhost:8080/"`
		serverFlagUsage  = `Server address in form "host:port"`
	)
	cfg := Config{}
	server := Server{
		Host: "localhost",
		Port: 8080,
	}
	cfg.BaseURL = "http://localhost:8080/"
	cfg.Logger.Level = "Info"
	cfg.Logger.Format = logger.Text
	cfg.FilePath = "data.json"

	flag.Var(&server, "a", serverFlagUsage)
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, baseURLFlagUsage)
	flag.StringVar(&cfg.FilePath, "f", cfg.FilePath, "File path storage")

	flag.Parse()

	if serverAddress, ok := os.LookupEnv(AddressKEY); ok {
		if err := server.UnmarshalText([]byte(serverAddress)); err != nil {
			return nil, err
		}
	}

	if baseURL, ok := os.LookupEnv(BaseURLKEY); ok {
		cfg.BaseURL = baseURL
	}

	if filePath, ok := os.LookupEnv(FilePathKEY); ok {
		cfg.FilePath = filePath
	}

	cfg.Server = server

	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid BaseURL %q: %v", cfg.BaseURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid BaseURL %q: must include scheme and host, e.g. http://localhost:8080/", cfg.BaseURL)
	}

	urlHost := u.Hostname()
	urlPortStr := u.Port()

	if urlPortStr == "" {
		return nil, fmt.Errorf(
			"invalid BaseURL %q: port must be specified explicitly, e.g. http://%s:%d/",
			cfg.BaseURL, cfg.Server.Host, cfg.Server.Port,
		)
	}

	urlPort, err := strconv.Atoi(urlPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid BaseURL %q: bad port %q: %v", cfg.BaseURL, urlPortStr, err)
	}

	if urlHost != cfg.Server.Host || urlPort != cfg.Server.Port {
		return nil, fmt.Errorf(
			"BaseURL %q does not match server address %s:%d; they must point to the same host and port",
			cfg.BaseURL, cfg.Server.Host, cfg.Server.Port,
		)
	}

	log.Printf("GetConfig: %s", &cfg)
	return &cfg, nil
}
