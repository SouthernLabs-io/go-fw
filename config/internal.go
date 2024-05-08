package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/sync"
)

// findConfigFile calls the recursiveFileFinder, to find a config file in Test model.
// If the file is not found, an os.ErrNotExist will be returned.
func findConfigFile(fileName string) (string, error) {
	if testing.Testing() {
		return recursiveFileFinder(fileName, "")
	}

	abs, err := filepath.Abs(fileName)
	if err != nil {
		return "", err
	}
	err = statFile(abs)
	if err != nil {
		return "", err
	}
	return abs, err
}

// statFile will check if the file exists.
func statFile(filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return errors.NewUnknownf("failed to stat file: %s, error: %w", filePath, err)
		}
		return os.ErrNotExist
	}
	return nil
}

func absFile(relFilePath string) (string, error) {
	filePath, err := filepath.Abs(relFilePath)
	if err != nil {
		return "", errors.NewUnknownf("failed to get absolute path for file: %s, error: %w", relFilePath, err)
	}
	return filePath, nil
}

// recursiveFileFinder will traverse the filesystem looking for a file with the given name.
// The algorithm to search is as follows:
//
//  1. Check if the file exists in a sub-folder called test.
//  2. Check if the file exists in current folder.
//  3. Check if the current folder has a go.mod file, if not, then step one level up and repeat the process.
//
// An os.ErrNotExist will be returned if it was not found.
func recursiveFileFinder(fileName string, prefix string) (string, error) {
	// 1. Check if the file exists in a sub-folder called test.
	filePath, err := absFile(filepath.Join(prefix, "test", fileName))
	if err != nil {
		return "", err
	}

	if err = statFile(filePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		// 2. Check if the file exists in current folder.
		filePath, err = absFile(filepath.Join(prefix, fileName))
		if err != nil {
			return "", err
		}
		if err = statFile(filePath); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return "", err
			}

			// 3. Check if the current folder has a go.mod file, if not, then step one level up and repeat the process.
			filePath, err = absFile(filepath.Join(prefix, "go.mod"))
			if err != nil {
				return "", err
			}
			if err = statFile(filePath); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return "", err
				}
				// go.mod does not exist in the current folder, step one level up and repeat the process.
				// But first check if we are at the root of the filesystem.
				if string(filepath.Separator) == filepath.Base(filePath) {
					return "", os.ErrNotExist
				}
				return recursiveFileFinder(fileName, "../"+prefix)
			} else {
				// go.mod exists in the current folder, but the file was not found.
				return "", os.ErrNotExist
			}
		}
	}

	// File found!
	return filePath, nil
}

var loadDotEnv = sync.OnceFunc(func() {
	dotEnvPath, err := findConfigFile(".env")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			panic(errors.NewUnknownf("failed to find dot env file, error: %w", err))
		}
	} else {
		println("Loading dot env file: ", dotEnvPath)
		err = godotenv.Load(dotEnvPath)
		if err != nil {
			panic(errors.NewUnknownf("failed to load dot env file: %s, error: %w", dotEnvPath, err))
		}
	}
})

var loadConfigYaml = sync.OnceValue(func() []byte {
	yamlConfPath, err := findConfigFile("config.yaml")
	if err != nil {
		panic(errors.NewUnknownf("failed to read config.yaml, error: %w", err))
	}

	println("Loading config file: ", yamlConfPath)
	yamlBytes, err := os.ReadFile(yamlConfPath)
	if err != nil {
		panic(errors.NewUnknownf("failed to read config.yaml, error: %w", err))
	}
	return yamlBytes
})

// FIXME: document the process of loading the config
func loadConfig[T any](conf *T, preprocess func(confMap map[string]any)) {
	loadDotEnv()
	yamlBytes := loadConfigYaml()

	// Unmarshal to the config struct
	// We use yaml -> map -> struct, because mapstructure will compare key names using strings.SameFold, which is case insensitive.
	confMap := make(map[string]any)
	err := yaml.Unmarshal(yamlBytes, &confMap)
	if err != nil {
		panic(errors.NewUnknownf("failed to unmarshal config to map, error: %w", err))
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Squash:           true,
		Result:           conf,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.TextUnmarshallerHookFunc(),
		),
	})
	if err != nil {
		panic(errors.NewUnknownf("failed to create decoder: %w", err))
	}
	err = decoder.Decode(confMap)
	if err != nil {
		panic(errors.NewUnknownf("failed to decode config map to struct: %T, error: %w", conf, err))
	}

	// Convert the config to a map. This time it will have all the keys, including those that where not specified in the config.yaml
	// We use struct -> json -> map, because mapstructure will create maps with custom values instead of the underlying types.
	confMap = make(map[string]any)
	jsonBytes, err := json.Marshal(conf)
	if err != nil {
		panic(errors.NewUnknownf("failed to marshal struct to yaml, error: %w", err))
	}
	err = json.Unmarshal(jsonBytes, &confMap)
	if err != nil {
		panic(errors.NewUnknownf("failed to unmarshal struct to map, error: %w", err))
	}

	// We use slog because config can't depend on log
	logger := slog.Default()

	// Parse current Environment variables
	envMap := make(map[string]string)
	for _, envPair := range os.Environ() {
		key, val, found := strings.Cut(envPair, "=")
		if !found {
			logger.Warn(fmt.Sprintf("[%T] Failed to parse env pair: %s, skipping it", *conf, envPair))
			continue
		}
		envMap[strings.ToLower(key)] = val
	}

	// Bind environment variables to the config map
	var bindEnvVars func(acc string, m map[string]any)
	bindEnvVars = func(acc string, m map[string]any) {
		for key, val := range m {
			switch v := val.(type) {
			case map[string]any:
				bindEnvVars(acc+strings.ToLower(key)+"_", v)
			default:
				envKey := acc + strings.ToLower(key)
				if envVal, ok := envMap[envKey]; ok {
					logger.Info(fmt.Sprintf("[%T] Using env key: %s", *conf, envKey))
					m[key] = envVal
				}
			}
		}
	}
	bindEnvVars("", confMap)

	if preprocess != nil {
		preprocess(confMap)
	}

	// Create a decoder with all the necessary hooks and decode the map to the conf struct
	decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Squash:           true,
		Result:           conf,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.TextUnmarshallerHookFunc(),
		),
	})
	if err != nil {
		panic(errors.NewUnknownf("failed to create decoder: %w", err))
	}
	err = decoder.Decode(confMap)
	if err != nil {
		panic(errors.NewUnknownf("failed to decode config map to struct: %T, error: %w", conf, err))
	}
}

func loadSecrets(conf RootConfig, secretsMgr SecretsManager) func(map[string]any) {
	return func(confMap map[string]any) {
		if conf.Env.Type == EnvTypeTest {
			return
		}
		ctx := context.Background()

		var traverse func(string, map[string]any)
		traverse = func(prefix string, m map[string]any) {
			for key, val := range m {
				key = prefix + key
				switch v := val.(type) {
				case map[string]any:
					traverse(key+".", v)
				case string:
					if !strings.HasPrefix(v, "<secret") || !strings.HasSuffix(v, ">") {
						continue
					}

					var smKey, secret string
					var err error
					if v == "<secret>" {
						smKey = key
						secret, err = secretsMgr.GetSecret(ctx, smKey)
					} else if v[7] == ':' {
						smKey = v[8 : len(v)-1]
						if smKey == "" {
							panic(errors.Newf(
								errors.ErrCodeBadState,
								"invalid secret value: %s, for key: %s, provide a custom key or remove ':'",
								v,
								key,
							))
						}
						secret, err = secretsMgr.GetSecretVerbatim(ctx, smKey)
					} else {
						panic(errors.Newf(errors.ErrCodeBadState, "invalid secret value: %s, for key: %s", v, key))
					}
					if err != nil {
						panic(errors.NewUnknownf("could not load secret: %s, error: %w", smKey, err))
					}
					m[key] = secret
				}
			}
		}
		traverse("", confMap)
	}
}
