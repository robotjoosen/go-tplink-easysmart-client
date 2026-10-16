package gui

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	// monitoring
	stateRegex           = `state:\[(.*)\]`
	linkStatusRegex      = `link_status:\[(.*)\]`
	linkStatusLabelRegex = `link_info=new Array\(([a-zA-Z" ,0-9]*)\)`
	packetsRegex         = `pkts:\[(.*)\]`
	maxPortsRegex        = `max_port_num = (.*);`

	// device info
	deviceInfoRegex = `".*"`

	// qos
	portNumberRegex = `var portNumber = (\d+);`
	bcInfoRegex     = `bcInfo\s*=\s*new Array\(([^)]*)\)`

	// cable diagnostics
	cableStateRegex  = `var cablestate = \[(.*?)\];`
	cableLengthRegex = `var cablelength = \[(.*?)\];`
)

func parseMaxPorts(html string) int {
	regex := regexp.MustCompile(maxPortsRegex)
	matches := regex.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return 0
	}

	raw := matches[0][1]
	i, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}

	return i
}

func parseLinkLabels(html string, limit int) []string {
	regex := regexp.MustCompile(linkStatusLabelRegex)
	matches := regex.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return make([]string, 0)
	}

	raw := matches[0][1]
	raw = strings.Replace(raw, `"`, ``, -1)

	return strings.Split(raw, ",")
}

func parseDeviceInfo(html string) []string {
	regex := regexp.MustCompile(deviceInfoRegex)
	matches := regex.FindAllStringSubmatch(html, 7)
	if len(matches) == 0 {
		return make([]string, 0)
	}

	output := make([]string, 0)
	for _, m := range matches {
		output = append(output, strings.Replace(m[0], `"`, ``, -1))
	}

	return output
}

func parseIntList(html, pattern string, limit int) []int {
	regex := regexp.MustCompile(pattern)
	matches := regex.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return make([]int, 0)
	}

	raw := matches[0][1]
	list := strings.Split(raw, ",")

	if len(list) < limit {
		return []int{}
	}

	output := make([]int, limit)
	for c := range output {
		i, err := strconv.Atoi(strings.TrimSpace(list[c]))
		if err != nil {
			return []int{}
		}

		output[c] = i
	}

	return output
}

// chunk splits a flat list into rows of n values; a list that does not hold
// exactly rows*n values yields no rows.
func chunk(list []int, n int) [][]int {
	if len(list) == 0 || len(list)%n != 0 {
		return [][]int{}
	}

	rows := make([][]int, 0, len(list)/n)
	for i := 0; i < len(list); i += n {
		rows = append(rows, list[i:i+n])
	}

	return rows
}

func parsePackets(html string, limit int) [][]int {
	return chunk(parseIntList(html, packetsRegex, limit*4), 4)
}

// parsePortSpeeds returns one row per port: ingress rate, egress rate and LAG id.
func parsePortSpeeds(html string, limit int) [][]int {
	return chunk(parseIntList(html, bcInfoRegex, limit*3), 3)
}

func parsePortNumber(html string) int {
	m := regexp.MustCompile(portNumberRegex).FindStringSubmatch(html)
	if m == nil {
		return 0
	}

	n, _ := strconv.Atoi(m[1])

	return n
}

func parseCableState(html string, limit int) []int {
	return parseIntList(html, cableStateRegex, limit)
}

func parseCableLength(html string, limit int) []int {
	return parseIntList(html, cableLengthRegex, limit)
}

func cableStateLabel(n int) string {
	switch n {
	case 0:
		return "No Cable"
	case 1:
		return "Normal"
	case 2:
		return "Open"
	case 3:
		return "Short"
	case 4:
		return "Open & Short"
	case 5:
		return "Cross Cable"
	case -1:
		return "Untested"
	default:
		return "Others"
	}
}
