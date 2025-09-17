package rosacli

import (
	"bytes"
)

type ImageMirrorService interface {
	CreateImageMirror(flags ...string) (bytes.Buffer, error)
	DeleteImageMirror(flags ...string) (bytes.Buffer, error)
	ListImageMirror(flags ...string) (bytes.Buffer, error)
	EditImageMirror(flags ...string) (bytes.Buffer, error)
	ReflectImageMirrorList(result bytes.Buffer) (isasl ImageMirrorList, err error)
}

type imageMirrorService struct {
	ResourcesService

	imageMirror map[string][]string
}

// Struct for the 'rosa list image-mirror' output for hosted-cp clusters
type ImageMirror struct {
	ID      string `json:"ID,omitempty"`
	Type    string `json:"TYPE,omitempty"`
	Source  string `json:"SOURCE,omitempty"`
	Mirrors string `json:"MIRRORS,omitempty"`
}
type ImageMirrorList struct {
	ImageMirrors []*ImageMirror `json:"ImageMirrors,omitempty"`
}

func (i *imageMirrorService) CreateImageMirror(flags ...string) (bytes.Buffer, error) {
	return i.client.Runner.
		Cmd("create", "image-mirror").
		CmdFlags(flags...).
		Run()
}

func (i *imageMirrorService) DeleteImageMirror(flags ...string) (bytes.Buffer, error) {
	return i.client.Runner.
		Cmd("delete", "image-mirror").
		CmdFlags(flags...).
		Run()
}

func (i *imageMirrorService) ListImageMirror(flags ...string) (bytes.Buffer, error) {
	return i.client.Runner.
		Cmd("list", "image-mirror").
		CmdFlags(flags...).
		Run()
}

func (i *imageMirrorService) EditImageMirror(flags ...string) (bytes.Buffer, error) {
	return i.client.Runner.
		Cmd("edit", "image-mirror").
		CmdFlags(flags...).
		Run()
}

func NewImageMirrorService(client *Client) ImageMirrorService {
	return &imageMirrorService{
		ResourcesService: ResourcesService{
			client: client,
		},
		imageMirror: make(map[string][]string),
	}
}

// Pasrse the result of 'rosa list image-mirror' to ImageMirrorlList struct
func (i *imageMirrorService) ReflectImageMirrorList(result bytes.Buffer) (iml ImageMirrorList, err error) {
	iml = ImageMirrorList{}
	theMap := i.client.Parser.TableData.Input(result).Parse().Output()
	for _, imItem := range theMap {
		im := &ImageMirror{}
		err = MapStructure(imItem, im)
		if err != nil {
			return
		}
		iml.ImageMirrors = append(iml.ImageMirrors, im)
	}
	return iml, err
}

func (iml ImageMirrorList) GetImageMirrorById(id string) (im ImageMirror) {
	for _, v := range iml.ImageMirrors {
		if v.ID == id {
			return *v
		}
	}
	return
}

func (iml ImageMirrorList) GetImageMirrorBySource(source string) (im ImageMirror) {
	for _, v := range iml.ImageMirrors {
		if v.Source == source {
			return *v
		}
	}
	return
}
