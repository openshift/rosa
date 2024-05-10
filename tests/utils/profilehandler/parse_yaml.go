package profilehandler

import (
	"log"
	"os"
	"path"

	"gopkg.in/yaml.v3"
)

type profiles struct {
	Profiles []*Profile `yaml:"profiles,omitempty"`
}

func ParseProfiles(profilesDir string) map[string]*Profile {
	files, err := os.ReadDir(profilesDir)
	if err != nil {
		log.Fatal(err)
	}

	profileMap := make(map[string]*Profile)
	for _, file := range files {
		yfile, err := os.ReadFile(path.Join(profilesDir, file.Name()))
		if err != nil {
			log.Fatal(err)
		}

		p := new(profiles)
		err = yaml.Unmarshal(yfile, &p)
		if err != nil {
			log.Fatal(err)
		}

		for _, theProfile := range p.Profiles {
			profileMap[theProfile.Name] = theProfile
		}

	}

	return profileMap
}

func ParseProfilesByFile(profileLocation string) map[string]*Profile {

	profileMap := make(map[string]*Profile)

	yfile, err := os.ReadFile(profileLocation)
	if err != nil {
		log.Fatal(err)
	}

	p := new(profiles)
	err = yaml.Unmarshal(yfile, &p)
	if err != nil {
		log.Fatal(err)
	}

	for _, theProfile := range p.Profiles {
		profileMap[theProfile.Name] = theProfile
	}

	return profileMap
}

func GetProfile(profileName string, profilesDir string) *Profile {
	profileMap := ParseProfiles(profilesDir)

	if _, exist := profileMap[profileName]; !exist {
		log.Fatalf("Can not find the profile %s in %s\n", profileName, profilesDir)
	}

	return profileMap[profileName]
}
