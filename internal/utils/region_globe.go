package utils

import (
	"math"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	regionGlobeWidth  = 36
	regionGlobeHeight = 18
	// RegionGlobePulseFrames controls how many spinner ticks pass before the marker changes intensity.
	RegionGlobePulseFrames = 8
	earthMapWidth          = 120
	earthMapHeight         = 60
)

var (
	deploymentTargetPattern = regexp.MustCompile(`(?i)cloud provider '([^']+)' and region '([^']+)'`)
	regionGlobeOceanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#172033"))
	regionGlobeLandStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#3F4B5F"))
	regionGlobeOrbStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#D9F99D")).Background(lipgloss.Color("#14532D")).Bold(true)
	regionGlobeOrbDimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")).Bold(true)
	regionGlobeTipStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#D1FAE5"))
	regionGlobeTipBorder    = lipgloss.NewStyle().Foreground(lipgloss.Color("#65A30D")).Bold(true)
)

var earthBitmap = []string{
	"                                                                                                                        ",
	"                                                                                                                        ",
	"                                                                                                                        ",
	"                             # ####### #################                                    #                           ",
	"                       #    #   ### #################            ###                                                    ",
	"                      ###  ## ####       ############ #                        ##         ########        #####         ",
	"                  ## ###   #  ### ##      ###########                         #    #### ################   ###          ",
	"      ######## ###### #### # #  #  ###     #########              #######        # ## ##################################",
	" ### ###########################    ####   #####      #          ####### ###############################################",
	"      ########################       ##    ####                #### ####################################################",
	"      ### # #################      ##        #                ##### # ##########################################  ##    ",
	"                ##############     #####                   #     #  #######################################      ##     ",
	"                 ################ #######                # #   ###########################################      ##      ",
	"                  ########################                 ################################################             ",
	"                    ###################  ##                ################################################             ",
	"                   ################### #                    ##########  ####  ############################              ",
	"                   ##################                    ##### ##  ###    ### ##########################                ",
	"                   #################                     ###       # ######## ######################  #    #            ",
	"                    ###############                       #  ###       ##############################  #  #             ",
	"                     #############                        ######        #############################                   ",
	"                       ######## #                        ############################################                   ",
	"                      # ####     #                      ##################### #######################                   ",
	"                       # ###      #                    ################# ######    #################                    ",
	"                         ###  #   #                    ################## ######     ####  #####                        ",
	"                          #####   # #                  ################## #####      ###    ####                        ",
	"                             ####                      ################### ###       ##      ####   #                   ",
	"                               #    #                  ####################           #      # ##                       ",
	"                                #  #####                #####################         #      # #     ##                 ",
	"                                   ######                #### ###############          #      #    #                    ",
	"                                   ########                     ############                 ##   ##                    ",
	"                                  #########                     ###########                   #  ####                   ",
	"                                  #############                 ##########                    ##### #     ##            ",
	"                                 ################                ########                                  ## #         ",
	"                                  ###############                #########                         ## #    # #          ",
	"                                   #############                 #########                                              ",
	"                                   ############                  #########  #                         # ##  #           ",
	"                                     ##########                 #########  ##                        ########           ",
	"                                     ##########                  #######   ##                      ###########     #    ",
	"                                     ########                    #######   #                      #############         ",
	"                                     #######                     ######                           ##############        ",
	"                                     #######                      #####                            #############        ",
	"                                     ######                       ####                             ###   ######         ",
	"                                    #####                                                                  ####       # ",
	"                                    #####                                                                              #",
	"                                    ###                                                                      #        # ",
	"                                    ###                                                                             ##  ",
	"                                    ##                                                                                  ",
	"                                   ##                                                                                   ",
	"                                    ##                                                                                  ",
	"                                                                                                                        ",
	"                                                                                                                        ",
	"                                                                                                                        ",
	"                                       #                                                                                ",
	"                                      #                                #  ##########   ########################         ",
	"                                   #####                 ########################## #################################   ",
	"                  # ## #   #############              #############################################################     ",
	"        ## #########################             ##################################################################     ",
	"           ######################## #  #  ##     #################################################################      ",
	"    ##################################################################################################################  ",
	"########################################################################################################################",
}

type regionGlobePoint struct {
	lat float64
	lon float64
}

// ParseDeploymentTarget extracts cloud provider and region from deploy progress text.
func ParseDeploymentTarget(text string) (provider, region string, ok bool) {
	matches := deploymentTargetPattern.FindStringSubmatch(text)
	if len(matches) != 3 {
		return "", "", false
	}
	return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2]), true
}

// RenderRegionGlobe renders a compact globe with a highlighted deployment region.
func RenderRegionGlobe(region string, width int, blinkOn bool) string {
	return RenderRegionGlobeWithProvider("", region, width, blinkOn)
}

// RenderRegionGlobeWithProvider renders a compact globe with cloud provider and deployment region labels.
func RenderRegionGlobeWithProvider(provider, region string, width int, blinkOn bool) string {
	region = strings.TrimSpace(region)
	if region == "" {
		return ""
	}
	if width <= 0 {
		width = regionGlobeWidth
	}

	lines := renderRegionGlobe(provider, region, blinkOn)

	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = lipgloss.NewStyle().MaxWidth(width).Render(line)
		}
	}
	return strings.Join(lines, "\n")
}

func renderRegionGlobe(provider, region string, blinkOn bool) []string {
	target := regionPoint(region)
	angle := target.lon
	markerX, markerY := projectRegionPoint(target, angle)
	marker := '●'
	if blinkOn {
		marker = '◉'
	}

	rows := make([][]rune, regionGlobeHeight)
	for y := range rows {
		rows[y] = make([]rune, regionGlobeWidth)
		for x := range rows[y] {
			rows[y][x] = ' '
		}
	}

	aspectRatio := 2.1
	centerX := float64(regionGlobeWidth) / 2
	centerY := float64(regionGlobeHeight) / 2
	radius := math.Min(centerX/1.1, centerY*aspectRatio/1.1)
	for y := 0; y < regionGlobeHeight; y++ {
		for x := 0; x < regionGlobeWidth; x++ {
			dx := float64(x) - centerX
			dy := (float64(y) - centerY) * aspectRatio
			distance := math.Sqrt(dx*dx + dy*dy)
			if distance > radius {
				continue
			}

			nx := dx / radius
			ny := dy / radius
			nzSq := 1 - nx*nx - ny*ny
			if nzSq < 0 {
				continue
			}
			nz := math.Sqrt(nzSq)
			lat := math.Asin(-ny) * 180 / math.Pi
			lon := normalizeLongitude(math.Atan2(nx, nz)*180/math.Pi + angle)
			rows[y][x] = globeTextureChar(lon, lat)
		}
	}
	if markerY >= 0 && markerY < len(rows) && markerX >= 0 && markerX < len(rows[markerY]) {
		rows[markerY][markerX] = marker
	}

	return renderRegionGlobeRows(rows, markerY, regionGlobeTooltip(provider, region))
}

func globeTextureChar(lon, lat float64) rune {
	if !isRegionGlobeLand(lon, lat) {
		return '.'
	}
	return '#'
}

func renderRegionGlobeRows(rows [][]rune, markerY int, tooltip []string) []string {
	rendered := make([]string, 0, len(rows))
	tooltipStartY := 0
	if len(tooltip) > 0 {
		tooltipStartY = spinnerClamp(markerY-len(tooltip)/2, 0, len(rows)-len(tooltip))
	}
	for y, row := range rows {
		var line strings.Builder
		for _, r := range row {
			switch r {
			case '◉':
				line.WriteString(regionGlobeOrbStyle.Render(string(r)))
			case '●':
				line.WriteString(regionGlobeOrbDimStyle.Render(string(r)))
			case '#':
				line.WriteString(regionGlobeLandStyle.Render(string(r)))
			case '.':
				line.WriteString(regionGlobeOceanStyle.Render(string(r)))
			default:
				line.WriteRune(r)
			}
		}
		renderedLine := strings.TrimRight(line.String(), " ")
		if strings.TrimSpace(renderedLine) == "" {
			continue
		}
		if tooltipLine, ok := regionGlobeTooltipLine(tooltip, tooltipStartY, markerY, y); ok {
			renderedLine = padRenderedRegionGlobeLine(renderedLine, regionGlobeWidth) + tooltipLine
		}
		rendered = append(rendered, renderedLine)
	}
	return rendered
}

func padRenderedRegionGlobeLine(line string, width int) string {
	padding := width - lipgloss.Width(line)
	if padding <= 0 {
		return line
	}
	return line + strings.Repeat(" ", padding)
}

func regionGlobeTooltip(provider, region string) []string {
	var values []string
	if provider = strings.TrimSpace(provider); provider != "" {
		values = append(values, provider)
	}
	if region = strings.TrimSpace(region); region != "" {
		values = append(values, region)
	}

	width := 0
	for _, value := range values {
		if lipgloss.Width(value) > width {
			width = lipgloss.Width(value)
		}
	}
	if width == 0 {
		return nil
	}

	lines := []string{
		regionGlobeTipBorder.Render("╭" + strings.Repeat("─", width+2) + "╮"),
	}
	for _, value := range values {
		padding := strings.Repeat(" ", width-lipgloss.Width(value))
		lines = append(lines, regionGlobeTipBorder.Render("│")+" "+regionGlobeTipStyle.Render(value+padding)+" "+regionGlobeTipBorder.Render("│"))
	}
	lines = append(lines, regionGlobeTipBorder.Render("╰"+strings.Repeat("─", width+2)+"╯"))
	return lines
}

func regionGlobeTooltipLine(tooltip []string, tooltipStartY, markerY, rowY int) (string, bool) {
	if len(tooltip) == 0 || rowY < tooltipStartY || rowY >= tooltipStartY+len(tooltip) {
		return "", false
	}
	connector := "  "
	if rowY == markerY {
		connector = regionGlobeTipBorder.Render(" ─")
	}
	return connector + tooltip[rowY-tooltipStartY], true
}

func projectRegionPoint(point regionGlobePoint, angle float64) (x, y int) {
	latRad := degreesToRadians(point.lat)
	lonRad := degreesToRadians(point.lon - angle)
	depth := math.Cos(latRad) * math.Cos(lonRad)
	if depth < 0 {
		return projectRegionPointFallback(point)
	}

	aspectRatio := 2.1
	centerX := float64(regionGlobeWidth) / 2
	centerY := float64(regionGlobeHeight) / 2
	radius := math.Min(centerX/1.1, centerY*aspectRatio/1.1)
	right := math.Cos(latRad) * math.Sin(lonRad)
	up := math.Sin(latRad)
	x = int(math.Round(centerX + right*radius))
	y = int(math.Round(centerY - up*radius/aspectRatio))
	return spinnerClamp(x, 0, regionGlobeWidth-1), spinnerClamp(y, 0, regionGlobeHeight-1)
}

func projectRegionPointFallback(point regionGlobePoint) (x, y int) {
	x = int(math.Round((point.lon + 180) / 360 * float64(regionGlobeWidth-1)))
	y = int(math.Round((90 - point.lat) / 180 * float64(regionGlobeHeight-1)))
	x = spinnerClamp(x, 0, regionGlobeWidth-1)
	y = spinnerClamp(y, 0, regionGlobeHeight-1)

	centerX := float64(regionGlobeWidth-1) / 2
	centerY := float64(regionGlobeHeight-1) / 2
	radiusX := centerX
	radiusY := centerY
	for !regionGlobeCellInside(x, y, centerX, centerY, radiusX, radiusY) {
		if float64(x) > centerX {
			x--
		} else if float64(x) < centerX {
			x++
		}
		if float64(y) > centerY {
			y--
		} else if float64(y) < centerY {
			y++
		}
	}
	return x, y
}

func regionGlobeCellInside(x, y int, centerX, centerY, radiusX, radiusY float64) bool {
	dx := (float64(x) - centerX) / radiusX
	dy := (float64(y) - centerY) / radiusY
	return dx*dx+dy*dy <= 1.0
}

func isRegionGlobeLand(lon, lat float64) bool {
	latNorm := (90 - lat) / 180
	lonNorm := (normalizeLongitude(lon) + 180) / 360
	y := spinnerClamp(int(latNorm*float64(earthMapHeight-1)), 0, earthMapHeight-1)
	x := spinnerClamp(int(lonNorm*float64(earthMapWidth-1)), 0, earthMapWidth-1)
	return earthBitmap[y][x] == '#'
}

func normalizeLongitude(lon float64) float64 {
	for lon > 180 {
		lon -= 360
	}
	for lon < -180 {
		lon += 360
	}
	return lon
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func regionPoint(region string) regionGlobePoint {
	r := strings.ToLower(strings.TrimSpace(region))
	r = strings.ReplaceAll(r, "_", "-")
	r = strings.ReplaceAll(r, " ", "")

	if point, ok := regionPointExact(r); ok {
		return point
	}

	switch {
	case strings.Contains(r, "brazil") || strings.HasPrefix(r, "sa-") || strings.HasPrefix(r, "southamerica-"):
		return regionGlobePoint{lat: -23, lon: -46}
	case strings.Contains(r, "southafrica") || strings.HasPrefix(r, "af-"):
		return regionGlobePoint{lat: -26, lon: 28}
	case strings.Contains(r, "australia") || strings.HasPrefix(r, "australia-"):
		return regionGlobePoint{lat: -33, lon: 151}
	case strings.Contains(r, "japan") || strings.Contains(r, "korea") || strings.Contains(r, "eastasia") || strings.HasPrefix(r, "ap-northeast") || strings.HasPrefix(r, "asia-east"):
		return regionGlobePoint{lat: 35, lon: 139}
	case strings.Contains(r, "india") || strings.HasPrefix(r, "ap-south") || strings.HasPrefix(r, "asia-south"):
		return regionGlobePoint{lat: 19, lon: 73}
	case strings.Contains(r, "singapore") || strings.Contains(r, "southeastasia") || strings.HasPrefix(r, "ap-southeast") || strings.HasPrefix(r, "asia-southeast"):
		return regionGlobePoint{lat: 1, lon: 104}
	case strings.Contains(r, "uae") || strings.Contains(r, "qatar") || strings.Contains(r, "israel") || strings.HasPrefix(r, "me-"):
		return regionGlobePoint{lat: 25, lon: 55}
	case strings.Contains(r, "canada") || strings.HasPrefix(r, "ca-") || strings.HasPrefix(r, "northamerica-"):
		return regionGlobePoint{lat: 45, lon: -73}
	case strings.Contains(r, "mexico"):
		return regionGlobePoint{lat: 19, lon: -99}
	case strings.Contains(r, "europe") || strings.Contains(r, "uk") || strings.Contains(r, "france") || strings.Contains(r, "germany") || strings.Contains(r, "sweden") || strings.Contains(r, "norway") || strings.Contains(r, "poland") || strings.HasPrefix(r, "eu-"):
		return regionGlobePoint{lat: 50, lon: 8}
	case strings.Contains(r, "eastus") || strings.Contains(r, "us-east") || strings.Contains(r, "useast"):
		return regionGlobePoint{lat: 38, lon: -78}
	case strings.Contains(r, "westus") || strings.Contains(r, "us-west") || strings.Contains(r, "uswest"):
		return regionGlobePoint{lat: 37, lon: -122}
	case strings.Contains(r, "centralus") || strings.Contains(r, "us-central") || strings.Contains(r, "uscentral") || strings.HasPrefix(r, "us-"):
		return regionGlobePoint{lat: 41, lon: -94}
	default:
		return regionGlobePoint{lat: 0, lon: 0}
	}
}

func regionPointExact(region string) (regionGlobePoint, bool) {
	points := map[string]regionGlobePoint{
		"af-south-1":         {lat: -26, lon: 28},
		"ap-east-1":          {lat: 22, lon: 114},
		"ap-northeast-1":     {lat: 35, lon: 139},
		"ap-northeast-2":     {lat: 37, lon: 127},
		"ap-northeast-3":     {lat: 34, lon: 135},
		"ap-south-1":         {lat: 19, lon: 73},
		"ap-south-2":         {lat: 17, lon: 78},
		"ap-southeast-1":     {lat: 1, lon: 104},
		"ap-southeast-2":     {lat: -33, lon: 151},
		"ap-southeast-3":     {lat: -6, lon: 107},
		"ap-southeast-4":     {lat: -37, lon: 145},
		"ca-central-1":       {lat: 45, lon: -73},
		"eu-central-1":       {lat: 50, lon: 8},
		"eu-central-2":       {lat: 47, lon: 8},
		"eu-north-1":         {lat: 59, lon: 18},
		"eu-north1":          {lat: 60, lon: 30},
		"eu-south-1":         {lat: 45, lon: 9},
		"eu-south-2":         {lat: 40, lon: -3},
		"eu-west-1":          {lat: 53, lon: -6},
		"eu-west-2":          {lat: 51, lon: 0},
		"eu-west-3":          {lat: 49, lon: 2},
		"il-central-1":       {lat: 32, lon: 35},
		"me-central-1":       {lat: 25, lon: 55},
		"me-south-1":         {lat: 26, lon: 50},
		"sa-east-1":          {lat: -23, lon: -46},
		"us-central1":        {lat: 41, lon: -94},
		"us-east-1":          {lat: 38, lon: -78},
		"us-east-2":          {lat: 40, lon: -83},
		"us-east1":           {lat: 34, lon: -81},
		"us-east4":           {lat: 39, lon: -77},
		"us-west-1":          {lat: 37, lon: -122},
		"us-west-2":          {lat: 45, lon: -122},
		"us-west1":           {lat: 45, lon: -122},
		"us-west2":           {lat: 34, lon: -118},
		"us-west3":           {lat: 40, lon: -112},
		"us-west4":           {lat: 36, lon: -115},
		"australiaeast":      {lat: -33, lon: 151},
		"australiasoutheast": {lat: -37, lon: 145},
		"brazilsouth":        {lat: -23, lon: -46},
		"canadacentral":      {lat: 45, lon: -73},
		"canadaeast":         {lat: 46, lon: -71},
		"centralindia":       {lat: 19, lon: 73},
		"centralus":          {lat: 41, lon: -94},
		"eastasia":           {lat: 22, lon: 114},
		"eastus":             {lat: 38, lon: -78},
		"eastus2":            {lat: 37, lon: -79},
		"francecentral":      {lat: 49, lon: 2},
		"germanywestcentral": {lat: 50, lon: 8},
		"japaneast":          {lat: 35, lon: 139},
		"koreacentral":       {lat: 37, lon: 127},
		"northeurope":        {lat: 53, lon: -6},
		"southafricanorth":   {lat: -26, lon: 28},
		"southcentralus":     {lat: 29, lon: -98},
		"southeastasia":      {lat: 1, lon: 104},
		"swedencentral":      {lat: 59, lon: 18},
		"uksouth":            {lat: 51, lon: 0},
		"westeurope":         {lat: 52, lon: 5},
		"westus":             {lat: 37, lon: -122},
		"westus2":            {lat: 45, lon: -122},
		"westus3":            {lat: 34, lon: -112},
	}
	point, ok := points[region]
	return point, ok
}
