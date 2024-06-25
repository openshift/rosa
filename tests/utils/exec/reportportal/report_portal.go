package reportportal

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/utils/log"
)

type TCStatus string

const (
	Passed   TCStatus = "passed"
	Skipped  TCStatus = "skipped"
	Failed   TCStatus = "failed"
	Panicked TCStatus = "panicked"
	Pending  TCStatus = "pending"
)

type Testsuites struct {
	XMLName  xml.Name     `xml:"testsuites"`
	Tests    int          `xml:"tests,attr"`
	Disabled int          `xml:"disabled,attr"`
	Errors   int          `xml:"errors,attr"`
	Failures int          `xml:"failures,attr"`
	Time     float64      `xml:"time,attr"`
	TS       []*Testsuite `xml:"testsuite"`
}

type Testsuite struct {
	XMLName    xml.Name   `xml:"testsuite"`
	Name       string     `xml:"name,attr"`
	Package    string     `xml:"package,attr"`
	Tests      int        `xml:"tests,attr"`
	Disabled   int        `xml:"disabled,attr"`
	Skipped    int        `xml:"skipped,attr"`
	Errors     int        `xml:"errors,attr"`
	Failures   int        `xml:"failures,attr"`
	Time       float64    `xml:"time,attr"`
	Timestamp  string     `xml:"timestamp,attr"`
	Properties Properties `xml:"properties"`
	TCs        []Testcase `xml:"testcase"`
}

type Properties struct {
	XMLName  xml.Name   `xml:"properties"`
	Property []Property `xml:"property"`
}

type Property struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}

type Testcase struct {
	XMLName   xml.Name        `xml:"testcase"`
	Name      string          `xml:"name,attr"`
	Classname string          `xml:"classname,attr"`
	Status    string          `xml:"status,attr"`
	Time      string          `xml:"time,attr"`
	FM        *FailureMessage `xml:"failure"`
	EM        *ErrorMessage   `xml:"error"`
}

type FailureMessage struct {
	XMLName  xml.Name `xml:"failure"`
	XMLValue string   `xml:",innerxml"`
	Message  string   `xml:"message,attr"`
	Type     string   `xml:"type,attr"`
}

type ErrorMessage struct {
	XMLName  xml.Name `xml:"error"`
	XMLValue string   `xml:",innerxml"`
	Message  string   `xml:"message,attr"`
	Type     string   `xml:"type,attr"`
}

func ParseJunitXML(fileName string) (*Testsuites, error) {
	testsuites := new(Testsuites)
	xmlFile, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal(xmlFile, testsuites)
	return testsuites, err
}

func GetExecutedTestCases(testsuite *Testsuite) ([]Testcase, []Testcase, []Testcase) {
	var passedTCList []Testcase
	var failedTCList []Testcase
	var panickedTCList []Testcase
	for _, tc := range testsuite.TCs {
		if tc.Name == "[BeforeSuite]" || tc.Name == "[AfterSuite]" {
			continue
		}

		switch tc.Status {
		case string(Skipped):
			continue
		case string(Passed):
			passedTCList = append(passedTCList, tc)
		case string(Failed):
			failedTCList = append(failedTCList, tc)
		case string(Panicked):
			panickedTCList = append(panickedTCList, tc)
		default:
			tcResult := PrintTestCase(tc)
			log.Logger.Warnf("Warning... Ohh!! There are testcases with outstanding status '%s': %s\n", tc.Status, tcResult.ID)
		}
	}

	return passedTCList, failedTCList, panickedTCList
}

func GenerateReportXMLFile() (int, int, map[string][]Testcase, map[string][]Testcase) {
	reportPortalDir := path.Join(config.Test.ArtifactDir, "junit")
	err := os.MkdirAll(reportPortalDir, 0777)
	if err != nil {
		panic(err)
	}

	issuedTCList := make(map[string][]Testcase)
	successedTCList := make(map[string][]Testcase)
	passedNum := 0
	issuedNum := 0

	xmlFileList := ListFiles(config.Test.ArtifactDir, ".xml")
	for _, xmlFile := range xmlFileList {
		xmlFilename := path.Base(xmlFile)
		xmlFilePrefix := strings.TrimSuffix(xmlFilename, ".xml")
		testsuites, err := ParseJunitXML(xmlFile)
		if err != nil {
			errMsg := fmt.Errorf("failed to parse file %s: %v", xmlFile, err)
			panic(errMsg)
		}

		for n, testsuite := range testsuites.TS {
			passedTCList, failedTCList, panickedTCList := GetExecutedTestCases(testsuite)
			executedNum := len(passedTCList) + len(failedTCList) + len(panickedTCList)
			passedNum = passedNum + len(passedTCList)
			issuedNum = issuedNum + len(failedTCList) + len(panickedTCList)
			issuedTCList[testsuite.Name] = append(issuedTCList[testsuite.Name], failedTCList...)
			issuedTCList[testsuite.Name] = append(issuedTCList[testsuite.Name], panickedTCList...)
			successedTCList[testsuite.Name] = append(successedTCList[testsuite.Name], passedTCList...)

			newTestsuite := &Testsuite{
				XMLName:    testsuite.XMLName,
				Name:       xmlFilePrefix,
				Package:    testsuite.Package,
				Tests:      executedNum,
				Disabled:   0,
				Skipped:    0,
				Errors:     len(panickedTCList),
				Failures:   len(failedTCList),
				Time:       testsuite.Time,
				Timestamp:  testsuite.Timestamp,
				Properties: testsuite.Properties,
			}

			newTestsuite.TCs = append(newTestsuite.TCs, passedTCList...)
			newTestsuite.TCs = append(newTestsuite.TCs, failedTCList...)
			newTestsuite.TCs = append(newTestsuite.TCs, panickedTCList...)
			xmlBody, _ := xml.MarshalIndent(newTestsuite, "", "	")

			reportFileName := fmt.Sprintf("import-%s-%d.xml", xmlFilePrefix, n+1)
			reportFile := path.Join(reportPortalDir, reportFileName)
			err := os.WriteFile(reportFile, xmlBody, 0666) // #nosec G306
			if err != nil {
				panic(err)
			}
		}
	}

	return passedNum, issuedNum, issuedTCList, successedTCList
}

func ListFiles(dir string, subfix string) []string {
	var Files []string
	fs, err := os.ReadDir(dir)
	if err != nil {
		panic(fmt.Sprintf("Failed to open the directory %s: %v\n", dir, err))
	}
	for _, f := range fs {
		if path.Ext(f.Name()) != subfix {
			continue
		}

		filename := path.Join(dir, f.Name())
		Files = append(Files, filename)
	}

	return Files
}

type TCResult struct {
	ID      string
	Title   string
	Message string
	Tags    []string
}

func PrintTestCase(tc Testcase) *TCResult {
	tcResult := new(TCResult)

	reg := regexp.MustCompile(`\[It\]\s(.*)\[id:(.*)\]\s\[(.*)\]`)
	params := reg.FindStringSubmatch(tc.Name)
	if len(params) > 0 {
		tcResult.Title = strings.TrimSpace(strings.Split(params[1], " -")[0])
		tcResult.ID = params[2]
		tcResult.Tags = strings.Split(params[3], ", ")
	} else {
		tcResult.Title = tc.Name
	}

	if tc.Status == string(Failed) {
		tcResult.Message = fmt.Sprintf("\t%s\n\t%s\n", tc.FM.Message, tc.FM.XMLValue)
	} else if tc.Status == string(Panicked) {
		tcResult.Message = fmt.Sprintf("\t%s\n\t%s\n", tc.EM.Message, tc.EM.XMLValue)
	}

	return tcResult
}

type ReportLogs struct {
	Total    int          `json:"total"`
	Passed   int          `json:"passed"`
	Failures int          `json:"failures"`
	Errors   int          `json:"errors"`
	Reports  []*ReportLog `json:"reports,omitempty"`
}

type ReportLog struct {
	TestSuite        string   `json:"testsuite,omitempty"`
	FailureScenarios []string `json:"failure_scenarios,omitempty"`
}

func GenerateReportLog() {
	reportPortalDir := path.Join(config.Test.ArtifactDir, "junit")
	reportLogFile := path.Join(config.Test.ArtifactDir, "e2e-test-results.json")

	reportLogs := new(ReportLogs)
	xmlFileList := ListFiles(reportPortalDir, ".xml")
	for _, xmlFile := range xmlFileList {
		testsuite := new(Testsuite)
		xmlFileBody, _ := os.ReadFile(xmlFile)
		err := xml.Unmarshal(xmlFileBody, testsuite)
		if err != nil {
			panic(fmt.Errorf("failed to parse file %s: %v", xmlFile, err))
		}

		reportLogs.Total += testsuite.Tests
		reportLogs.Errors += testsuite.Errors
		reportLogs.Failures += testsuite.Failures
		reportLogs.Passed += testsuite.Tests - testsuite.Errors - testsuite.Failures

		_, failedTCList, panickedTCList := GetExecutedTestCases(testsuite)
		failedTCList = append(failedTCList, panickedTCList...)

		var failureScenarios []string
		for _, issuedTC := range failedTCList {
			tcResult := PrintTestCase(issuedTC)
			tcName := fmt.Sprintf("OCP-%s: %s", tcResult.ID, tcResult.Title)
			failureScenarios = append(failureScenarios, tcName)
		}

		reportLog := &ReportLog{
			TestSuite:        testsuite.Name,
			FailureScenarios: failureScenarios,
		}
		reportLogs.Reports = append(reportLogs.Reports, reportLog)
	}

	jsonBody, _ := json.MarshalIndent(reportLogs, "", " ")
	err := os.WriteFile(reportLogFile, jsonBody, 0666) // #nosec G306
	if err != nil {
		panic(err)
	}
}
