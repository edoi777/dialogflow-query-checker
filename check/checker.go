package check

import (
	"github.com/yoichiro/dialogflow-query-checker/config"
	"github.com/yoichiro/dialogflow-query-checker/query"
	"container/list"
	"fmt"
	"regexp"
	"strings"
	"time"
)

func Execute(def *config.Definition) (*list.List, error) {
	results := list.New()
	for _, test := range def.Tests {
		actual, err := query.Execute(&test, def.ClientAccessToken, def.DefaultLanguage)
		if err != nil {
			return nil, err
		}
		displayResult(assertIntEquals(results, &test, "status.code", 200, actual.Status.Code))
		displayResult(assertStringEquals(results, &test,"action", test.Expect.Action, actual.Result.Action))
		displayResult(assertStringEquals(results, &test,"intentName", test.Expect.IntentName, actual.Result.Metadata.IntentName))
		actualContexts := make([]string, len(actual.Result.Contexts))
		for i, context := range actual.Result.Contexts {
			actualContexts[i] = context.Name
		}
		if test.Expect.Contexts != nil {
			displayResult(assertArrayEquals(results, &test,"contexts", test.Expect.Contexts, actualContexts))
		}
		displayResult(assertStringEquals(results, &test,"date", evaluateDateMacro(test.Expect.Parameters.Date, "2006-01-02"), actual.Result.Parameters.Date))
		displayResult(assertStringEquals(results, &test,"prefecture", test.Expect.Parameters.Prefecture, actual.Result.Parameters.Prefecture))
		displayResult(assertStringEquals(results, &test,"keyword", test.Expect.Parameters.Keyword, actual.Result.Parameters.Keyword))
		displayResult(assertStringEquals(results, &test,"event", test.Expect.Parameters.Event, actual.Result.Parameters.Event))
		if test.Expect.Speeches != nil {
			displayResult(assertByMultipleRegexps(results, &test, "speech", test.Expect.Speeches, actual.Result.Fulfillment.Speech))
		} else {
			re := regexp.MustCompile(evaluateDateMacro(test.Expect.Speech, "1月2日"))
			displayResult(assertByRegexp(results, &test,"speech", re, actual.Result.Fulfillment.Speech))
		}
	}
	return results, nil
}

func displayResult(result bool) {
	if result {
		fmt.Print(".")
	} else {
		fmt.Print("F")
	}
}

func assertIntEquals(results *list.List, test *config.Test, name string, expected int, actual int) bool {
	if expected != actual {
		results.PushBack(
			fmt.Sprintf("%s %s is not same. expected:%d actual:%d",
				test.CreatePrefix(), name, expected, actual))
		return false
	}
	return true
}

func assertStringEquals(results *list.List, test *config.Test, name string, expected string, actual string) bool {
	if expected != actual {
		results.PushBack(
			fmt.Sprintf("%s %s is not same. expected:%s actual:%s",
				test.CreatePrefix(), name, expected, actual))
		return false
	}
	return true
}

func assertArrayEquals(results *list.List, test *config.Test, name string, expected []string, actual []string) bool {
	if len(expected) != len(actual) {
		results.PushBack(
			fmt.Sprintf("%s The length of %s is not same. expected:%d actual:%d",
				test.CreatePrefix(), name, len(expected), len(actual)))
		return false
	}
	for _, e := range expected {
		if !contains(actual, e) {
			results.PushBack(fmt.Sprintf("%s does not contain %s", name, e))
			return false
		}
	}
	return true
}

func assertByRegexp(results *list.List, test *config.Test, name string, expected *regexp.Regexp, actual string) bool {
	if !expected.Match([]byte(actual)) {
		results.PushBack(
			fmt.Sprintf("%s %s does not match. expected:%s actual:%s",
				test.CreatePrefix(), name, expected, actual))
		return false
	}
	return true
}

func assertByMultipleRegexps(results *list.List, test *config.Test, name string, regexps []string, actual string) bool {
	for _, exp := range regexps {
		re := regexp.MustCompile(evaluateDateMacro(exp, "1月2日"))
		if re.Match([]byte(actual)) {
			return true
		}
	}
	f := func(x string) string {
		return fmt.Sprintf("\"%s\"", x)
	}
	results.PushBack(
		fmt.Sprintf("%s %s does not match. expected:%s actual:%s",
			test.CreatePrefix(), name, strings.Join(mapString(regexps, f), ", "), actual))
	return false
}

func mapString(x []string, f func(string) string) []string {
	r := make([]string, len(x))
	for i, e := range x {
		r[i] = f(e)
	}
	return r
}

func contains(array []string, s string) bool {
	for _, e := range array {
		if s == e {
			return true
		}
	}
	return false
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
