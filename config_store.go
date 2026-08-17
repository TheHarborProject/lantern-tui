package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const lanternConfigFilename = "lantern.config.json"

type ConfigStore struct {
	path string
}

func NewConfigStore(path string) *ConfigStore { return &ConfigStore{path: path} }

func NewCWDConfigStore() (*ConfigStore, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve current working directory: %w", err)
	}
	return NewConfigStore(filepath.Join(cwd, lanternConfigFilename)), nil
}

func (s *ConfigStore) Load() (*AuthoredConfig, bool, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		config := defaultAuthoredConfig()
		config.markNew()
		return config, false, nil
	}
	if err != nil {
		return nil, true, fmt.Errorf("read %s: %w", lanternConfigFilename, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		return nil, true, fmt.Errorf("invalid %s: %w", lanternConfigFilename, err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, true, fmt.Errorf("invalid %s: %w", lanternConfigFilename, err)
	}
	config, err := authoredConfigFromDocument(document)
	if err != nil {
		return nil, true, fmt.Errorf("invalid %s: %w", lanternConfigFilename, err)
	}
	return config, true, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("multiple JSON documents")
		}
		return err
	}
	return nil
}

func (s *ConfigStore) Save(config *AuthoredConfig) error {
	data, err := json.MarshalIndent(config.Document(), "", "  ")
	if err != nil {
		return fmt.Errorf("serialize config: %w", err)
	}
	data = append(data, '\n')
	if !json.Valid(data) {
		return fmt.Errorf("serialized config is not valid JSON")
	}

	directory := filepath.Dir(s.path)
	temporary, err := os.CreateTemp(directory, ".lantern.config.json-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err = temporary.Write(data); err == nil {
		err = temporary.Sync()
	}
	closeErr := temporary.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("replace %s: %w", lanternConfigFilename, err)
	}
	if directoryHandle, err := os.Open(directory); err == nil {
		_ = directoryHandle.Sync()
		_ = directoryHandle.Close()
	}
	config.markSaved()
	return nil
}
