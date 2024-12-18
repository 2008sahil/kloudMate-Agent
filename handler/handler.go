package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"sigs.k8s.io/yaml"
)

type CollectorManager struct {
	Cmd *exec.Cmd
}

func HandleConfig(body []byte) error {
	currentMap := map[string]interface{}{}
	if err := json.Unmarshal(body, &currentMap); err != nil {
		return err
	}

	base := map[string]interface{}{}
	data, err := ioutil.ReadFile("config/baseconfig.yaml")
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, &base); err != nil {
		return err
	}

	base = mergeMaps(base, currentMap)

	mergedData, err := yaml.Marshal(base)
	if err != nil {
		return err
	}
	mergedConfigPath := "config/tempconfig.yaml"

	if err := ioutil.WriteFile(mergedConfigPath, mergedData, 0644); err != nil {
		return err
	}

	isValid, err := validateConfig(mergedConfigPath)
	if err != nil {
		os.Remove(mergedConfigPath)
		return fmt.Errorf("error during configuration validation: %w", err)
	}

	if !isValid {
		os.Remove(mergedConfigPath)
		return fmt.Errorf("configuration validation failed")
	}

	newConfigPath := "config/finalconfig.yaml"

	if err := ioutil.WriteFile(newConfigPath, mergedData, 0644); err != nil {
		os.Remove(mergedConfigPath)
		return err
	}
	os.Remove(mergedConfigPath)

	return nil

}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if bv, ok := out[k]; ok {
			if bvMap, ok := bv.(map[string]interface{}); ok {
				if vMap, ok := v.(map[string]interface{}); ok {
					out[k] = mergeMaps(bvMap, vMap)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func validateConfig(configPath string) (bool, error) {
	cmd := exec.Command("./collector/collector", "validate", fmt.Sprintf("--config=%s", configPath))
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Validation failed: ", err)
		return false, nil
	}
	fmt.Println("Validation successfull ")
	return true, nil
}
