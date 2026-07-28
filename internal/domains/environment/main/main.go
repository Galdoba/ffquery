package main

import "github.com/Galdoba/ffquery/internal/domains/environment"

func main() {
	roots := []string{
		"IN",
		"PROGRESS",
		"DONE",
		"SCRIPTS",
		"EDIT",
		"ARCHIVE",
	}
	dr := environment.NewRegistry()

	dr.AddDirectories([]environment.Directory{
		//IN
		{
			Host: environment.Host{
				IP:   "//192.168.31.4",
				OS:   "linux",
				Name: "buffer",
			},
			Alias: "IN",
			PathFromPerspective: map[string]string{
				"buffer":      "/home/pemaltynov/IN/",
				"workstation": `\\192.168.31.4\buffer\IN`,
			},
		},
		//PROGRESS
		{
			Host: environment.Host{
				IP:   "",
				OS:   "",
				Name: "",
			},
			Alias:               "",
			PathFromPerspective: map[string]string{},
		},
	}...)
	dr.HostNames["//192.168.31.4"] = "workstation"
	dr.HostNames["//192.168.31.55"] = "buffer"
	dr.HostNames["//192.168.31.6"] = "dev-db"
}
