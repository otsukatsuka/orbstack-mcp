package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/otsukatsuka/orbstack-mcp/docker"
)

type containerInspectArgs struct {
	Container string `json:"container" jsonschema:"required,description=container name or ID to inspect"`
	Section   string `json:"section" jsonschema:"description=section to display: env ports volumes network all (default: all)"`
}

func handleContainerInspect(ctx context.Context, exec docker.Executor, args containerInspectArgs) (string, error) {
	if args.Container == "" {
		return "", fmt.Errorf("container name or ID is required")
	}

	section := args.Section
	if section == "" {
		section = "all"
	}

	out, err := exec.Exec(ctx, "inspect", args.Container)
	if err != nil {
		return "", err
	}

	// docker inspect returns a JSON array
	var inspectData []map[string]any
	if err := json.Unmarshal([]byte(out), &inspectData); err != nil {
		return "", fmt.Errorf("failed to parse inspect JSON: %w", err)
	}

	if len(inspectData) == 0 {
		return "", fmt.Errorf("no inspect data returned for container %s", args.Container)
	}

	data := inspectData[0]

	switch section {
	case "env":
		return formatEnvSection(data)
	case "ports":
		return formatPortsSection(data)
	case "volumes":
		return formatVolumesSection(data)
	case "network":
		return formatNetworkSection(data)
	case "all":
		return formatAllSection(out)
	default:
		return "", fmt.Errorf("unknown section %q: must be one of env, ports, volumes, network, all", section)
	}
}

func getNestedMap(data map[string]any, keys ...string) (map[string]any, bool) {
	current := data
	for _, key := range keys {
		val, ok := current[key]
		if !ok {
			return nil, false
		}
		m, ok := val.(map[string]any)
		if !ok {
			return nil, false
		}
		current = m
	}
	return current, true
}

func formatEnvSection(data map[string]any) (string, error) {
	config, ok := getNestedMap(data, "Config")
	if !ok {
		return "No Config section found.", nil
	}

	envRaw, ok := config["Env"]
	if !ok {
		return "No environment variables configured.", nil
	}

	envSlice, ok := envRaw.([]any)
	if !ok {
		return "No environment variables configured.", nil
	}

	var b strings.Builder
	b.WriteString("Environment Variables:\n")
	for _, e := range envSlice {
		if s, ok := e.(string); ok {
			b.WriteString(fmt.Sprintf("  %s\n", s))
		}
	}
	return b.String(), nil
}

func formatPortsSection(data map[string]any) (string, error) {
	var b strings.Builder
	b.WriteString("Port Bindings:\n")

	// HostConfig.PortBindings
	hostConfig, ok := getNestedMap(data, "HostConfig")
	if ok {
		if pbRaw, ok := hostConfig["PortBindings"]; ok {
			if pb, ok := pbRaw.(map[string]any); ok && len(pb) > 0 {
				for port, bindings := range pb {
					if bindSlice, ok := bindings.([]any); ok {
						for _, bind := range bindSlice {
							if bindMap, ok := bind.(map[string]any); ok {
								hostIP, _ := bindMap["HostIp"].(string)
								hostPort, _ := bindMap["HostPort"].(string)
								if hostIP == "" {
									hostIP = "0.0.0.0"
								}
								b.WriteString(fmt.Sprintf("  %s -> %s:%s\n", port, hostIP, hostPort))
							}
						}
					}
				}
			} else {
				b.WriteString("  No port bindings configured.\n")
			}
		} else {
			b.WriteString("  No port bindings configured.\n")
		}
	}

	// NetworkSettings.Ports
	netSettings, ok := getNestedMap(data, "NetworkSettings")
	if ok {
		if portsRaw, ok := netSettings["Ports"]; ok {
			if ports, ok := portsRaw.(map[string]any); ok && len(ports) > 0 {
				b.WriteString("\nExposed Ports:\n")
				for port, mappings := range ports {
					if mappings == nil {
						b.WriteString(fmt.Sprintf("  %s -> (not mapped)\n", port))
					} else if mapSlice, ok := mappings.([]any); ok {
						for _, m := range mapSlice {
							if mMap, ok := m.(map[string]any); ok {
								hostIP, _ := mMap["HostIp"].(string)
								hostPort, _ := mMap["HostPort"].(string)
								if hostIP == "" {
									hostIP = "0.0.0.0"
								}
								b.WriteString(fmt.Sprintf("  %s -> %s:%s\n", port, hostIP, hostPort))
							}
						}
					}
				}
			}
		}
	}

	return b.String(), nil
}

func formatVolumesSection(data map[string]any) (string, error) {
	mountsRaw, ok := data["Mounts"]
	if !ok {
		return "No volumes mounted.", nil
	}

	mountsSlice, ok := mountsRaw.([]any)
	if !ok || len(mountsSlice) == 0 {
		return "No volumes mounted.", nil
	}

	var b strings.Builder
	b.WriteString("Mounts:\n")
	for _, m := range mountsSlice {
		if mount, ok := m.(map[string]any); ok {
			mountType, _ := mount["Type"].(string)
			source, _ := mount["Source"].(string)
			destination, _ := mount["Destination"].(string)
			rw, _ := mount["RW"].(bool)
			mode := "rw"
			if !rw {
				mode = "ro"
			}
			b.WriteString(fmt.Sprintf("  [%s] %s -> %s (%s)\n", mountType, source, destination, mode))
		}
	}
	return b.String(), nil
}

func formatNetworkSection(data map[string]any) (string, error) {
	netSettings, ok := getNestedMap(data, "NetworkSettings")
	if !ok {
		return "No network settings found.", nil
	}

	networksRaw, ok := netSettings["Networks"]
	if !ok {
		return "No networks configured.", nil
	}

	networks, ok := networksRaw.(map[string]any)
	if !ok || len(networks) == 0 {
		return "No networks configured.", nil
	}

	var b strings.Builder
	b.WriteString("Networks:\n")
	for name, netRaw := range networks {
		b.WriteString(fmt.Sprintf("  %s:\n", name))
		if net, ok := netRaw.(map[string]any); ok {
			if ip, ok := net["IPAddress"].(string); ok && ip != "" {
				b.WriteString(fmt.Sprintf("    IP Address: %s\n", ip))
			}
			if gw, ok := net["Gateway"].(string); ok && gw != "" {
				b.WriteString(fmt.Sprintf("    Gateway:    %s\n", gw))
			}
			if mac, ok := net["MacAddress"].(string); ok && mac != "" {
				b.WriteString(fmt.Sprintf("    MAC:        %s\n", mac))
			}
		}
	}
	return b.String(), nil
}

func formatAllSection(rawJSON string) (string, error) {
	// Pretty-print the full inspect output
	var data any
	if err := json.Unmarshal([]byte(rawJSON), &data); err != nil {
		return rawJSON, nil
	}
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return rawJSON, nil
	}
	return string(pretty), nil
}

func registerContainerInspect(server *mcp.Server, exec docker.Executor) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "container_inspect",
		Description: "Get detailed container information. Optionally filter by section: env, ports, volumes, network, or all (default).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args containerInspectArgs) (*mcp.CallToolResult, any, error) {
		result, err := handleContainerInspect(ctx, exec, args)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return nil, result, nil
	})
}
