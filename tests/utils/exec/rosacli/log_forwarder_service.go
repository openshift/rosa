package rosacli

import (
	"bytes"
)

type LogForwarderService interface {
	DescribeLogForwarder(clusterID string, flags ...string) (bytes.Buffer, error)
	ListLogForwarder(clusterID string, flags ...string) (bytes.Buffer, error)
	CreateLogForwarder(clusterName string, flags ...string) (bytes.Buffer, error)
	DeleteLogForwarder(clusterID string, flags ...string) (bytes.Buffer, error)
	EditLogForwarder(clusterID string, flags ...string) (bytes.Buffer, error)
	ReflectLogForwarderList(result bytes.Buffer) (logForwardersList *LogForwarderList, err error)
}

type logForwarderService struct {
	ResourcesService
}

func NewLogForwarderService(client *Client) LogForwarderService {
	return &logForwarderService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

// Struct for the 'rosa list log-forwarders' output
type LogForwarderListItem struct {
	ID     string `yaml:"ID,omitempty"`
	Type   string `yaml:"TYPE,omitempty"`
	Status string `yaml:"STATUS,omitempty"`
}
type LogForwarderList struct {
	LogForwarders []*LogForwarderListItem `yaml:"LogForwarders,omitempty"`
}

func (l *logForwarderService) DescribeLogForwarder(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	describe := l.client.Runner.
		Cmd("describe", "log-forwarder").
		CmdFlags(combflags...)
	return describe.Run()
}

func (l *logForwarderService) ListLogForwarder(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	list := l.client.Runner.Cmd("list", "log-forwarders").CmdFlags(combflags...)
	return list.Run()
}

func (l *logForwarderService) CreateLogForwarder(clusterName string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterName}, flags...)
	createCommand := l.client.Runner.
		Cmd("create", "log-forwarder").
		CmdFlags(combflags...)
	output, err := createCommand.Run()
	return output, err
}

func (l *logForwarderService) DeleteLogForwarder(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	deleteCluster := l.client.Runner.
		Cmd("delete", "log-forwarder").
		CmdFlags(combflags...)
	return deleteCluster.Run()
}

func (l *logForwarderService) EditLogForwarder(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	editCluster := l.client.Runner.
		Cmd("edit", "log-forwarder").
		CmdFlags(combflags...)
	return editCluster.Run()
}

func (l *logForwarderService) ReflectLogForwarderList(result bytes.Buffer) (
	logForwardersList *LogForwarderList, err error) {
	logForwardersList = &LogForwarderList{}
	theMap := l.client.Parser.TableData.Input(result).Parse().Output()
	for _, item := range theMap {
		logForwarder := &LogForwarderListItem{}
		err = MapStructure(item, logForwarder)
		if err != nil {
			return logForwardersList, err
		}
		logForwardersList.LogForwarders = append(logForwardersList.LogForwarders, logForwarder)
	}
	return logForwardersList, err
}

func (logForwardsList LogForwarderList) GetLogForwarderByType(ltype string) (logForwarder *LogForwarderListItem) {
	for _, lfwd := range logForwardsList.LogForwarders {
		if lfwd.Type == ltype {
			return lfwd
		}
	}
	return
}
