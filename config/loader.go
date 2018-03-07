package config

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"os"
	"github.com/google/uuid"
	"errors"
	"fmt"
	"time"
	"strings"
	"reflect"
)

func LoadConfigurationFile(path string) (*Definition, error) {
	buf, err := loadFromFile(path)
	if err != nil {
		return nil, err
	}

	def, err := loadConfiguration(buf)
	if err != nil {
		return nil, err
	}

	return def, nil
}

func loadConfiguration(buf []byte) (*Definition, error) {
	def, err := parseFile(buf)
	if err != nil {
		return nil, err
	}

	err = preprocess(def)
	if err != nil {
		return nil, err
	}

	return def, nil
}

func loadFromFile(path string) ([]byte, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func parseFile(buf []byte) (*Definition, error) {
	var def Definition
	err := yaml.Unmarshal(buf, &def)
	if err != nil {
		return nil, err
	}
	return &def, err
}

func preprocess(def *Definition) error {
	determineClientAccessToken(def)
	determineSessionId(def)
	err := determineLanguageAndLocale(def)
	determineDateMacro(def)
	if err != nil {
		return err
	}
	determineServiceAccessToken(def)
	determineScore(def)
	return nil
}

func determineScore(def *Definition) {
	defaultScoreThreshold := def.DefaultScoreThreshold
	for i := range def.Tests {
		expect := &def.Tests[i].Expect
		if expect.ScoreThreshold == 0 {
			expect.ScoreThreshold = defaultScoreThreshold
		}
	}
}

func determineLanguageAndLocale(def *Definition) error {
	language := def.DefaultLanguage
	locale := def.DefaultLocale
	for i := range def.Tests {
		condition := &def.Tests[i].Condition
		// Determine language
		if condition.Language == "" || condition.Language == "inherit" {
			if language != "" {
				condition.Language = language
			} else {
				return errors.New(fmt.Sprintf("[%s] Cannot determine a language", def.Tests[i].CreatePrefix()))
			}
		} else {
			language = condition.Language
		}
		// Determine locale
		if condition.Locale == "" || condition.Locale == "inherit" {
			if locale != "" {
				condition.Locale = locale
			} else {
				return errors.New(fmt.Sprintf("[%s] Cannot determine a locale", def.Tests[i].CreatePrefix()))
			}
		} else {
			locale = condition.Locale
		}
	}
	return nil
}

func determineServiceAccessToken(def *Definition) {
	if def.DefaultServiceAccessToken == "" {
		def.DefaultServiceAccessToken = os.Getenv("DIALOGFLOW_SERVICE_ACCESS_TOKEN")
	}
	for i := range def.Tests {
		condition := &def.Tests[i].Condition
		if condition.ServiceAccessToken == "" {
			condition.ServiceAccessToken = def.DefaultServiceAccessToken
		}
	}
}

func determineDateMacro(def *Definition) {
	for i := range def.Tests {
		condition := &def.Tests[i].Condition
		condition.Query = evaluateDateMacro(condition.Query, def.DateMacroFormat)

		traverseMapAndEvaluateDateMacro(&def.Tests[i].Expect.Parameters)

		expect := &def.Tests[i].Expect
		expect.Speech = evaluateDateMacro(expect.Speech, def.DateMacroFormat)
		for i, v := range expect.Speeches {
			expect.Speeches[i] = evaluateDateMacro(v, def.DateMacroFormat)
		}
	}
}

func traverseMapAndEvaluateDateMacro(m *map[interface{}]interface{}) {
	for key := range *m {
		value := (*m)[key]
		if value != nil && reflect.TypeOf(value).String() == "string" {
			(*m)[key] = evaluateDateMacro(value.(string), "2006-01-02")
		} else if value != nil && strings.HasPrefix(reflect.TypeOf(value).String(), "map") {
			child := value.(map[interface{}]interface{})
			traverseMapAndEvaluateDateMacro(&child)
		}
	}
}

func evaluateDateMacro(s string, layout string) string {
	t := time.Now()
	today := t.Format(layout)
	t = t.AddDate(0, 0, 1)
	tomorrow := t.Format(layout)
	result := strings.Replace(s, "${date.tomorrow}", tomorrow, -1)
	result = strings.Replace(result, "${date.today}", today, -1)
	return result
}

func determineSessionId(def *Definition) {
	sessionId := issueSessionId()
	for i := range def.Tests {
		condition := &def.Tests[i].Condition
		if condition.SessionId == "" || condition.SessionId == "inherit" {
			condition.SessionId = sessionId
		} else if condition.SessionId == "new" {
			sessionId = issueSessionId()
			condition.SessionId = sessionId
		} else {
			// Use the specified value as a Session ID.
		}
	}
}

func issueSessionId() string {
	return uuid.Must(uuid.NewRandom()).String()
}

func determineClientAccessToken(def *Definition) {
	if def.ClientAccessToken == "" {
		def.ClientAccessToken = os.Getenv("DIALOGFLOW_CLIENT_ACCESS_TOKEN")
	}
}