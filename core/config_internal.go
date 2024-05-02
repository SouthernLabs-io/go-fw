package core

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/iancoleman/strcase"
	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cast"
	"github.com/spf13/viper"

	"github.com/southernlabs-io/go-fw/errors"
)

// findConfigFile calls the recursiveFileFinder, to find a config file. If the file is not found, an os.ErrNotExist will be returned.
func findConfigFile(fileName string) (string, error) {
	return recursiveFileFinder(fileName, "")
}

// recursiveFileFinder will traverse the tree upwards until a go.mod file is found in test mode. An os.ErrNotExist will be returned if
// it was not found.
func recursiveFileFinder(fileName string, prefix string) (string, error) {
	filePath, err := filepath.Abs(prefix + fileName)
	if err != nil {
		return "", err
	}
	if filePath == "/"+fileName {
		return "", os.ErrNotExist
	}
	if _, err = os.Stat(filePath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", errors.NewUnknownf("failed to stat file: %s, error: %w", filePath, err)
		}
		// This asserts the application is not test mode before it walks up the tree to find a go.mod file.
		if !testing.Testing() {
			return "", os.ErrNotExist
		}
		if _, err = os.Stat(prefix + "go.mod"); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return "", errors.NewUnknownf("failed to stat file: go.mod, error: %w", err)
			}
			return recursiveFileFinder(fileName, "../"+prefix)
		}
		return "", os.ErrNotExist
	}
	return filePath, nil
}

func loadConfig(conf any, preprocess func(v *viper.Viper, structKeys map[string]bool)) {
	v := viper.New()

	dotEnvFile := ".env"
	if testing.Testing() {
		dotEnvFile = ".env.test"
	}

	dotEnvPath, err := findConfigFile(dotEnvFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			panic(errors.NewUnknownf("failed to find dot env file, error: %w", err))
		}
	} else {
		err = godotenv.Load(dotEnvPath)
		if err != nil {
			panic(errors.NewUnknownf("failed to load dot env file: %s, error: %w", dotEnvPath, err))
		}
	}

	yamlConfPath, err := findConfigFile("config.yaml")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			panic(errors.NewUnknownf("failed to read config.yaml, error: %w", err))
		}
	} else {
		v.SetConfigFile(yamlConfPath)
		err = v.ReadInConfig()
		if err != nil {
			panic(errors.NewUnknownf("failed to load config file: %s, error: %w", yamlConfPath, err))
		}
	}

	structKeys, err := getStructKeys(v, conf)
	if err != nil {
		panic(errors.NewUnknownf("failed to get struct keys for: %T, error: %w", conf, err))
	}

	err = bindStructKeys(v, structKeys)
	if err != nil {
		panic(errors.NewUnknownf("failed to bind struct keys from os.Environ(), error: %w", err))
	}

	if preprocess != nil {
		preprocess(v, structKeys)
	}

	err = v.Unmarshal(
		conf,
		viper.DecodeHook(
			mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.StringToSliceHookFunc(","),
				mapstructure.TextUnmarshallerHookFunc(),
			),
		),
	)
	if err != nil {
		panic(errors.NewUnknownf("failed to unmarshal config to struct: %T, error: %w", conf, err))
	}
}

func loadSecrets(conf CoreConfig) func(*viper.Viper, map[string]bool) {
	return func(v *viper.Viper, structKeys map[string]bool) {
		if conf.Env.Type == EnvTypeTest {
			return
		}
		lf := NewLoggerFactory(conf)
		secretsMgr := NewAWSSecretsManager(conf, NewAWSConfig(conf), lf)
		ctx := context.Background()
		for key := range structKeys {
			val := v.GetString(key)
			if !strings.HasPrefix(val, "<secret") || !strings.HasSuffix(val, ">") {
				continue
			}

			var smKey, secret string
			var err error
			if val == "<secret>" {
				smKey = strcase.ToKebab(key)
				secret, err = secretsMgr.GetSecret(ctx, smKey)
			} else if val[7] == ':' {
				smKey = val[8 : len(val)-1]
				secret, err = secretsMgr.GetSecretVerbatim(ctx, smKey)
			} else {
				panic(errors.Newf(errors.ErrCodeBadArgument, "invalid secret key: %s", key))
			}
			if err != nil {
				panic(errors.NewUnknownf("could not load secret: %s, error: %w", smKey, err))
			}
			v.Set(key, secret)
		}
	}
}

func bindStructKeys(v *viper.Viper, structKeys map[string]bool) error {
	for key := range structKeys {
		if err := v.BindEnv(key, strcase.ToScreamingSnake(strings.ReplaceAll(key, ".", "_"))); err != nil {
			return err
		}
	}

	return nil
}

func getStructKeys(v *viper.Viper, input any) (map[string]bool, error) {
	envKeysMap := map[string]any{}
	if err := mapstructure.Decode(input, &envKeysMap); err != nil {
		return nil, err
	}

	return flattenAndMergeMap(v, map[string]bool{}, envKeysMap, ""), nil
}

// flattenAndMergeMap recursively flattens the given map into a map[string]bool
// of key paths (used as a set, easier to manipulate than a []string):
//   - each path is merged into a single key string, delimited with v.keyDelim
//   - if a path is shadowed by an earlier value in the initial shadow map,
//     it is skipped.
//
// The resulting set of paths is merged to the given shadow set at the same time.
// Copied from Viper source code
func flattenAndMergeMap(v *viper.Viper, shadow map[string]bool, m map[string]interface{}, prefix string) map[string]bool {
	if shadow != nil && prefix != "" && shadow[prefix] {
		// prefix is shadowed => nothing more to flatten
		return shadow
	}
	if shadow == nil {
		shadow = make(map[string]bool)
	}

	vType := reflect.ValueOf(v).Elem()
	keyDelimField := vType.FieldByName("keyDelim")

	var m2 map[string]interface{}
	if prefix != "" {
		prefix += keyDelimField.String() //NOTE: this is an unexported field, so we use reflection: v.keyDelim
	}
	for k, val := range m {
		fullKey := prefix + k
		switch tv := val.(type) {
		case map[string]interface{}:
			m2 = tv
		case map[interface{}]interface{}:
			m2 = cast.ToStringMap(val)
		default:
			// immediate value
			shadow[fullKey] = true //Note: Original code was using toLower which breaks our intention of using camelCase to snake
			continue
		}
		// recursively merge to shadow map
		shadow = flattenAndMergeMap(v, shadow, m2, fullKey)
	}
	return shadow
}
