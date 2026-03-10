package buildinfo

var (
	BuildTime string
	Stable    string
	Branch    string
)

func IsStable() bool {
	return Stable == "true"
}

func BranchValue() string {
	if Branch == "" {
		return "unknown"
	}
	return Branch
}
