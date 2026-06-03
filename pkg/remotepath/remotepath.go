// Package remotepath provides a universal representation of file paths
// in a heterogeneous environment (Windows/Linux, different machines).
// It allows converting a path from one machine's representation to another's.
// TODO: обдумать вывести этот пакет в отдельное приложение или нет.
package remotepath

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Resource represents a logical resource (network share, mount point).
// Each resource has a unique identifier.
type Resource struct {
	ID      string                 // e.g. "buffer", "root"
	Mapping map[string]MachineView // key: machine ID (hostname or alias)
}

// MachineView describes how a resource is seen on a specific machine.
type MachineView struct {
	// OS for which this path is applicable: "windows", "linux", or "" (any)
	OS string
	// Local root of the resource.
	// For Windows: UNC prefix, e.g. `\\192.168.0.101\buffer`
	// For Linux: absolute mount path, e.g. `/home/user/IN`
	Root string
	// Optional alternative network path, can be used for URLs.
	NetworkRoot string // e.g. `smb://192.168.101/buffer`
}

// Config describes all known machines and resources.
type Config struct {
	// Current machine identifier (set by the application).
	CurrentMachine string `json:"current_machine"`
	// All known machines: key = machine ID (e.g. "workstation", "fileserver", "linux-box").
	Machines map[string]Machine `json:"machines"`
	// List of resources.
	Resources map[string]Resource `json:"resources"`
}

// Machine describes properties of a machine.
type Machine struct {
	OS       string `json:"os"`       // "windows" or "linux"
	Hostname string `json:"hostname"` // optional, for information
}

// Path represents a universal path: resource + relative path (Unix-style).
type Path struct {
	ResourceID string
	RelPath    string // always with forward slashes, no leading slash
}

// Manager handles conversions between machines.
type Manager struct {
	config Config
}

// NewManager creates a manager with the given configuration.
func NewManager(cfg Config) *Manager {
	return &Manager{config: cfg}
}

// NewPathFromString creates a universal path from a string, auto-detecting the resource
// using the current machine (from config.CurrentMachine).
func (m *Manager) NewPathFromString(rawPath string) (*Path, error) {
	return m.NewPathFromStringForMachine(rawPath, m.config.CurrentMachine)
}

// NewPathFromStringForMachine creates a universal path from a string, explicitly specifying
// on which machine this path is represented.
func (m *Manager) NewPathFromStringForMachine(rawPath, machineID string) (*Path, error) {
	// Normalize separators to Unix style
	normalized := filepath.ToSlash(rawPath)

	machine, ok := m.config.Machines[machineID]
	if !ok {
		return nil, fmt.Errorf("unknown machine: %s", machineID)
	}

	// Find a resource whose root matches the beginning of the path
	for resID, res := range m.config.Resources {
		for mid, view := range res.Mapping {
			if mid != machineID {
				continue
			}
			if view.OS != "" && view.OS != machine.OS {
				continue
			}
			rootNorm := filepath.ToSlash(view.Root)
			if after, ok0 := strings.CutPrefix(normalized, rootNorm); ok0 {
				rel := after
				rel = strings.TrimPrefix(rel, "/")
				return &Path{
					ResourceID: resID,
					RelPath:    rel,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("no resource mapping found for path %q on machine %s", rawPath, machineID)
}

// ToLocalPath returns the local path for the specified machine (using its OS separator).
func (p *Path) ToLocalPath(m *Manager, machineID string) (string, error) {
	res, ok := m.config.Resources[p.ResourceID]
	if !ok {
		return "", fmt.Errorf("unknown resource: %s", p.ResourceID)
	}
	machine, ok := m.config.Machines[machineID]
	if !ok {
		return "", fmt.Errorf("unknown machine: %s", machineID)
	}
	view, ok := res.Mapping[machineID]
	if !ok {
		return "", fmt.Errorf("no mapping for resource %s on machine %s", p.ResourceID, machineID)
	}
	if view.OS != "" && view.OS != machine.OS {
		return "", fmt.Errorf("view OS %s doesn't match machine OS %s", view.OS, machine.OS)
	}
	root := view.Root
	rel := filepath.FromSlash(p.RelPath)
	return filepath.Join(root, rel), nil
}

// ToUNC returns the UNC representation (only for Windows machines).
// Returns an error if the target machine is not Windows.
func (p *Path) ToUNC(m *Manager, machineID string) (string, error) {
	machine, ok := m.config.Machines[machineID]
	if !ok {
		return "", fmt.Errorf("unknown machine: %s", machineID)
	}
	if machine.OS != "windows" {
		return "", fmt.Errorf("machine %s is not Windows, cannot produce UNC", machineID)
	}
	path, err := p.ToLocalPath(m, machineID)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(path, `\\`) {
		return "", fmt.Errorf("root for machine %s is not UNC: %s", machineID, path)
	}
	return path, nil
}

// ToURL returns a file:// URL for the specified machine.
// If the resource defines NetworkRoot, that is used instead (e.g. smb://).
func (p *Path) ToURL(m *Manager, machineID string) (string, error) {
	res, ok := m.config.Resources[p.ResourceID]
	if !ok {
		return "", fmt.Errorf("unknown resource: %s", p.ResourceID)
	}
	view, ok := res.Mapping[machineID]
	if !ok {
		return "", fmt.Errorf("no mapping for resource %s on machine %s", p.ResourceID, machineID)
	}
	if view.NetworkRoot != "" {
		base := strings.TrimSuffix(view.NetworkRoot, "/")
		relEncoded := url.PathEscape(p.RelPath)
		return base + "/" + relEncoded, nil
	}
	// Fallback: build file:// URL from local path
	localPath, err := p.ToLocalPath(m, machineID)
	if err != nil {
		return "", err
	}
	u := &url.URL{
		Scheme: "file",
		Path:   localPath,
	}
	if m.config.Machines[machineID].OS == "windows" && strings.HasPrefix(localPath, `\\`) {
		// Convert \\server\share\path -> file://server/share/path
		trimmed := strings.TrimPrefix(localPath, `\\`)
		parts := strings.SplitN(trimmed, `\`, 2)
		if len(parts) == 2 {
			u.Host = parts[0]
			u.Path = "/" + filepath.ToSlash(parts[1])
		} else {
			u.Path = filepath.ToSlash(localPath)
		}
	} else {
		u.Path = filepath.ToSlash(localPath)
	}
	return u.String(), nil
}

// ToCurrentOSPath returns the path for the current OS, using the default machine.
func (p *Path) ToCurrentOSPath(m *Manager) (string, error) {
	return p.ToLocalPath(m, m.config.CurrentMachine)
}

// ConvertToString converts the universal path to a string for the target machine.
func (p *Path) ConvertToString(m *Manager, targetMachineID string) (string, error) {
	return p.ToLocalPath(m, targetMachineID)
}

// ConvertBetweenMachines converts a string path from the source machine to the target machine.
func (m *Manager) ConvertBetweenMachines(rawPath, srcMachine, dstMachine string) (string, error) {
	p, err := m.NewPathFromStringForMachine(rawPath, srcMachine)
	if err != nil {
		return "", err
	}
	return p.ToLocalPath(m, dstMachine)
}

// ConvertToUNCFromMachine converts a path from the source machine to UNC for the target (must be Windows).
func (m *Manager) ConvertToUNCFromMachine(rawPath, srcMachine, dstMachine string) (string, error) {
	p, err := m.NewPathFromStringForMachine(rawPath, srcMachine)
	if err != nil {
		return "", err
	}
	return p.ToUNC(m, dstMachine)
}

// ConvertToURLFromMachine converts a path from the source machine to a URL for the target.
func (m *Manager) ConvertToURLFromMachine(rawPath, srcMachine, dstMachine string) (string, error) {
	p, err := m.NewPathFromStringForMachine(rawPath, srcMachine)
	if err != nil {
		return "", err
	}
	return p.ToURL(m, dstMachine)
}

// LoadConfigFromJSON loads configuration from JSON data and returns a Manager.
func LoadConfigFromJSON(data []byte, currentMachine string) (*Manager, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.CurrentMachine = currentMachine
	return NewManager(cfg), nil
}

// Example JSON configuration (backticks used instead of double quotes for illustration):
/*
{
  `current_machine`: `workstation`,
  `machines`: {
    `workstation`: { `os`: `windows`, `hostname`: `WS-01` },
    `fileserver`: { `os`: `windows`, `hostname`: `FS01` },
    `linux-box`: { `os`: `linux`, `hostname`: `ubuntu` }
  },
  `resources`: {
    `buffer`: {
      `mapping`: {
        `workstation`: { `os`: `windows`, `root`: `\\\\192.168.31.4\\buffer`, `network_root`: `smb://192.168.31.4/buffer` },
        `fileserver`: { `os`: `windows`, `root`: `\\\\FS01\\shared_buffer`, `network_root`: `smb://FS01/shared_buffer` },
        `linux-box`: { `os`: `linux`, `root`: `/home/pemaltynov/IN`, `network_root`: `smb://192.168.31.4/buffer` }
      }
    },
    `root`: {
      `mapping`: {
        `workstation`: { `os`: `windows`, `root`: `\\\\192.168.31.4\\root`, `network_root`: `smb://192.168.31.4/root` },
        `fileserver`: { `os`: `windows`, `root`: `\\\\FS01\\root_share` },
        `linux-box`: { `os`: `linux`, `root`: `/mnt/pemaltynov/ROOT`, `network_root`: `smb://192.168.31.4/root` }
      }
    }
  }
}
*/
