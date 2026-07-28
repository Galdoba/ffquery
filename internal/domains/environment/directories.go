package environment

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
)

const ()

// Host describes a machine by its IP, OS, and optionally a human-friendly name.
// IP is the primary identifier; Name is auxiliary (for logging/display).
type Host struct {
	IP   string `json:"ip"`
	OS   string `json:"os"`
	Name string `json:"name,omitempty"` // optional, for display only
}

// LocalHost returns the Host describing the current machine.
// It populates IP and OS; Name is set to the OS hostname (for convenience).
func LocalHost() (Host, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return Host{}, fmt.Errorf("failed to get hostname: %w", err)
	}
	ip := getLocalIP()
	return Host{
		IP:   ip,
		OS:   runtime.GOOS,
		Name: hostname,
	}, nil
}

// getLocalIP returns the first non-loopback IPv4 address, or "127.0.0.1" if none found.
func getLocalIP() string {
	const defaultLocalIP = "127.0.0.1"
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return defaultLocalIP
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return defaultLocalIP
}

// Directory represents a logical directory with host-specific paths.
// Host holds the primary machine info (at least IP and OS).
// PathFromPerspective maps an IP address (string) to the corresponding path on that host.
type Directory struct {
	Host                Host              `json:"host"`
	Alias               string            `json:"alias"`
	PathFromPerspective map[string]string `json:"path_from_perspective"` // key = IP
}

// DirRegistry stores directories grouped by alias, plus a mapping from IP to display name.
type DirRegistry struct {
	Dirs      map[string][]Directory `json:"dirs"`
	HostNames map[string]string      `json:"host_names"` // IP -> human-readable name
}

func NewRegistry() *DirRegistry {
	dr := DirRegistry{}
	dr.Dirs = make(map[string][]Directory)
	return &dr
}

// GetPathLocal returns the local path for the given alias, appending any subDirs.
// It uses the IP of the local host to look up the path.
func (dr *DirRegistry) GetPathLocal(rootAlias string, subDirs ...string) (string, error) {
	host, err := LocalHost()
	if err != nil {
		return "", fmt.Errorf("failed to get local host: %w", err)
	}
	return dr.GetPathRemote(host, rootAlias, subDirs...)
}

// GetPathRemote returns the path for the given alias on the specified host,
// appending any subDirs. The path is looked up by the host's IP address.
// The path separator style matches the host OS stored in the Directory.
func (dr *DirRegistry) GetPathRemote(host Host, rootAlias string, subDirs ...string) (string, error) {
	dirs, ok := dr.Dirs[rootAlias]
	if !ok {
		return "", fmt.Errorf("alias %q not found", rootAlias)
	}
	for _, dir := range dirs {
		basePath, exists := dir.PathFromPerspective[host.IP]
		if !exists {
			continue
		}
		parts := append([]string{basePath}, subDirs...)
		joined := filepath.Join(parts...)
		return normalizePath(dir.Host.OS, joined), nil
	}
	return "", fmt.Errorf("path for IP %q and alias %q not found", host.IP, rootAlias)
}

// normalizePath converts the given path to the format expected by the host OS.
func normalizePath(hostOS, path string) string {
	const windowsOS = "windows"
	switch hostOS {
	case windowsOS:
		return filepath.FromSlash(path)
	default:
		return filepath.ToSlash(path)
	}
}

// Load reads a DirRegistry from a JSON file.
func Load(pathToJson string) (*DirRegistry, error) {
	data, err := os.ReadFile(pathToJson)
	if err != nil {
		return nil, err
	}
	var dr DirRegistry
	if err := json.Unmarshal(data, &dr); err != nil {
		return nil, err
	}
	// Ensure HostNames is never nil for convenience.
	if dr.HostNames == nil {
		dr.HostNames = make(map[string]string)
	}
	return &dr, nil
}

// Save writes the DirRegistry to a JSON file with indentation.
func (dr *DirRegistry) Save(pathToJson string) error {
	const fileMode = 0644
	data, err := json.MarshalIndent(dr, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pathToJson, data, fileMode)
}

// AddDirectories adds one or more Directories to the registry.
// Each Directory must have a non‑empty Alias field – it is used as the grouping key.
// If any Directory has an empty Alias, an error is returned and no changes are made.
func (dr *DirRegistry) AddDirectories(dirs ...Directory) error {
	if dr.Dirs == nil {
		dr.Dirs = make(map[string][]Directory)
	}
	for _, d := range dirs {
		if d.Alias == "" {
			return fmt.Errorf("directory has empty Alias field")
		}
	}
	for _, d := range dirs {
		dr.Dirs[d.Alias] = append(dr.Dirs[d.Alias], d)
	}
	return nil
}

// RemoveDirectory removes all directories whose primary host IP matches the given host.
// Returns an error if the alias does not exist or no directory for that IP is found.
func (dr *DirRegistry) RemoveDirectory(alias string, host Host) error {
	dirs, ok := dr.Dirs[alias]
	if !ok {
		return fmt.Errorf("alias %q not found", alias)
	}
	newDirs := make([]Directory, 0, len(dirs))
	removed := false
	for _, d := range dirs {
		if d.Host.IP == host.IP {
			removed = true
			continue
		}
		newDirs = append(newDirs, d)
	}
	if !removed {
		return fmt.Errorf("no directory for IP %q under alias %q", host.IP, alias)
	}
	if len(newDirs) == 0 {
		delete(dr.Dirs, alias)
	} else {
		dr.Dirs[alias] = newDirs
	}
	return nil
}

// GetDirectories returns all directories registered for the given alias.
func (dr *DirRegistry) GetDirectories(alias string) ([]Directory, error) {
	dirs, ok := dr.Dirs[alias]
	if !ok {
		return nil, fmt.Errorf("alias %q not found", alias)
	}
	return dirs, nil
}

// SetPerspective adds or updates the path for a given host under the alias,
// using the host's IP as the key. If no directory exists for the alias,
// a new one is created with the given host as the primary Host.
// It also updates the HostNames mapping if host.Name is non-empty.
func (dr *DirRegistry) SetPerspective(alias string, host Host, path string) {
	if dr.Dirs == nil {
		dr.Dirs = make(map[string][]Directory)
	}
	// Update hostnames mapping for display purposes.
	if dr.HostNames == nil {
		dr.HostNames = make(map[string]string)
	}
	if host.Name != "" {
		dr.HostNames[host.IP] = host.Name
	}

	dirs := dr.Dirs[alias]
	// Look for an existing directory whose primary host IP matches.
	for i, d := range dirs {
		if d.Host.IP == host.IP {
			// Update the perspective map for this directory.
			if dr.Dirs[alias][i].PathFromPerspective == nil {
				dr.Dirs[alias][i].PathFromPerspective = make(map[string]string)
			}
			dr.Dirs[alias][i].PathFromPerspective[host.IP] = path
			// Also update the host OS if needed (e.g., if OS changed).
			dr.Dirs[alias][i].Host.OS = host.OS
			return
		}
	}
	// No matching primary host found; create a new directory entry.
	newDir := Directory{
		Host:  host,
		Alias: alias,
		PathFromPerspective: map[string]string{
			host.IP: path,
		},
	}
	dr.Dirs[alias] = append(dirs, newDir)
}

// RemovePerspective removes the path for the given host (by IP) from the directory.
// If the perspective map becomes empty, the entire directory entry is removed.
func (dr *DirRegistry) RemovePerspective(alias string, host Host) error {
	dirs, ok := dr.Dirs[alias]
	if !ok {
		return fmt.Errorf("alias %q not found", alias)
	}
	for i, d := range dirs {
		if _, exists := d.PathFromPerspective[host.IP]; exists {
			delete(d.PathFromPerspective, host.IP)
			if len(d.PathFromPerspective) == 0 {
				// Remove the entire directory entry.
				dr.Dirs[alias] = append(dirs[:i], dirs[i+1:]...)
				if len(dr.Dirs[alias]) == 0 {
					delete(dr.Dirs, alias)
				}
			} else {
				dr.Dirs[alias][i] = d
			}
			return nil
		}
	}
	return fmt.Errorf("IP %q not found in alias %q", host.IP, alias)
}

// ResolveHost returns the display name for the given IP, or the IP itself if no name is known.
// This is useful for logging or generating user-friendly messages.
func (dr *DirRegistry) ResolveHost(ip string) string {
	if dr.HostNames == nil {
		return ip
	}
	if name, ok := dr.HostNames[ip]; ok && name != "" {
		return name
	}
	return ip
}
